package main

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/agrinman/kr"
	"github.com/golang/groupcache/lru"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrTimeout = errors.New("Request timed out")
var ErrNotPaired = errors.New("Phone not paired")

//	Message queued during send
type SendQueued struct {
	error
}

func (err *SendQueued) Error() string {
	return fmt.Sprintf("SendQueued: " + err.error.Error())
}

//	Network-related error during send
type SendError struct {
	error
}

func (err *SendError) Error() string {
	return fmt.Sprintf("SendError: " + err.error.Error())
}

//	Network-related error during receive
type RecvError struct {
	error
}

func (err *RecvError) Error() string {
	return fmt.Sprintf("RecvError: " + err.error.Error())
}

//	Unrecoverable error, this request will always fail
type ProtoError struct {
	error
}

func (err *ProtoError) Error() string {
	return fmt.Sprintf("ProtoError: " + err.error.Error())
}

type EnclaveClientI interface {
	kr.Transport
	Pair() (pairing *kr.PairingSecret, err error)
	IsPaired() bool
	Unpair()
	Start() (err error)
	Stop() (err error)
	RequestMe(longTimeout bool) (*kr.MeResponse, error)
	GetCachedMe() *kr.Profile
	RequestSignature(kr.SignRequest) (*kr.SignResponse, error)
	RequestNoOp() error
}

type EnclaveClient struct {
	sync.Mutex
	kr.Transport
	Timeouts
	kr.Persister
	pairingSecret               *kr.PairingSecret
	requestCallbacksByRequestID *lru.Cache
	ackedRequestIDs             *lru.Cache
	outgoingQueue               [][]byte
	snsEndpointARN              *string
	cachedMe                    *kr.Profile
	bt                          BluetoothDriverI
}

func (ec *EnclaveClient) Pair() (pairingSecret *kr.PairingSecret, err error) {
	ec.Lock()
	defer ec.Unlock()

	err = ec.generatePairing()
	if err != nil {
		return
	}
	err = ec.activatePairing()
	if err != nil {
		return
	}

	pairingSecret = ec.pairingSecret

	return
}

func (ec *EnclaveClient) Unpair() {
	ec.Lock()
	defer ec.Unlock()
	if ec.pairingSecret != nil {
		ec.unpair(ec.pairingSecret, true)
	}
	return
}

func (ec *EnclaveClient) IsPaired() bool {
	ps := ec.getPairingSecret()
	if ps == nil {
		return false
	}
	return ps.IsPaired()
}

func (ec *EnclaveClient) generatePairing() (err error) {
	if ec.pairingSecret != nil {
		ec.unpair(ec.pairingSecret, true)
	}
	ec.Persister.DeleteMe()
	ec.Persister.DeletePairing()

	pairingSecret, err := kr.GeneratePairingSecret()
	if err != nil {
		log.Error(err)
		return
	}

	err = ec.Transport.Setup(pairingSecret)
	if err != nil {
		log.Error(err)
		return
	}

	//	erase any existing pairing
	ec.pairingSecret = pairingSecret
	ec.outgoingQueue = [][]byte{}

	savePairingErr := ec.Persister.SavePairing(pairingSecret)
	if savePairingErr != nil {
		log.Error("error saving pairing:", savePairingErr.Error())
	}
	return
}

func (ec *EnclaveClient) unpair(pairingSecret *kr.PairingSecret, sendUnpairRequest bool) (err error) {
	if ec.pairingSecret == nil || !ec.pairingSecret.Equals(pairingSecret) {
		return
	}
	ec.deactivatePairing(pairingSecret)
	ec.cachedMe = nil
	ec.pairingSecret = nil
	ec.Persister.DeleteMe()
	ec.Persister.DeletePairing()
	if sendUnpairRequest {
		func() {
			unpairRequest, err := kr.NewRequest()
			if err != nil {
				log.Error("error creating request:", err)
				return
			}
			unpairRequest.UnpairRequest = &kr.UnpairRequest{}
			unpairJson, err := json.Marshal(unpairRequest)
			if err != nil {
				log.Error("error creating request:", err)
				return
			}
			go ec.sendMessage(pairingSecret, unpairJson, false)
		}()
	}
	return
}

func (ec *EnclaveClient) deactivatePairing(pairingSecret *kr.PairingSecret) (err error) {
	if ec.bt != nil {
		oldBtUUID, uuidErr := pairingSecret.DeriveUUID()
		if uuidErr == nil {
			btErr := ec.bt.RemoveService(oldBtUUID)
			if btErr != nil {
				log.Error("error removing bluetooth service:", btErr.Error())
			}
		}
	}
	return
}

func (ec *EnclaveClient) activatePairing() (err error) {
	if ec.bt != nil {
		if ec.pairingSecret != nil {
			btUUID, uuidErr := ec.pairingSecret.DeriveUUID()
			if uuidErr != nil {
				err = uuidErr
				log.Error(err)
				return
			}
			err = ec.bt.AddService(btUUID)
			if err != nil {
				log.Error(err)
				return
			}
		}
	}
	return
}
func (ec *EnclaveClient) Stop() (err error) {
	ec.Lock()
	defer ec.Unlock()
	if ec.pairingSecret != nil {
		ec.deactivatePairing(ec.pairingSecret)
	}
	if ec.bt != nil {
		ec.bt.Stop()
	}
	return
}

func (ec *EnclaveClient) Start() (err error) {
	ec.Lock()
	defer ec.Unlock()
	loadedPairing, loadErr := ec.Persister.LoadPairing()
	if loadErr == nil && loadedPairing != nil {
		ec.pairingSecret = loadedPairing
	} else {
		log.Notice("pairing not loaded:", loadErr)
	}

	if loadedMe, loadMeErr := ec.Persister.LoadMe(); loadMeErr == nil {
		ec.cachedMe = &loadedMe
	} else {
		log.Notice("me not loaded:", loadErr)
	}

	bt, err := NewBluetoothDriver()
	if err != nil {
		log.Error("error starting bluetooth driver:", err)
	} else {
		ec.bt = bt
		go func() {
			readChan, err := ec.bt.ReadChan()
			if err != nil {
				log.Error("error retrieving bluetooth read channel:", err)
				return
			}
			for ciphertext := range readChan {
				err = ec.handleCiphertext(ciphertext, "bluetooth")
				if err != nil {
					log.Error("error reading bluetooth channel:", err)
				}
			}
		}()
	}

	ec.activatePairing()
	return
}

func (ec *EnclaveClient) getPairingSecret() (pairingSecret *kr.PairingSecret) {
	ec.Lock()
	defer ec.Unlock()
	pairingSecret = ec.pairingSecret
	return
}

func (ec *EnclaveClient) GetCachedMe() (me *kr.Profile) {
	ec.Lock()
	defer ec.Unlock()
	me = ec.cachedMe
	return
}

func (ec *EnclaveClient) postEvent(category string, action string, label *string, value *uint64) {
	ps := ec.getPairingSecret()
	if ps != nil {
		tID := ps.GetTrackingID()
		if tID != nil {
			go kr.Analytics{}.PostEvent(*tID, category, action, label, value)
		}
	}
}

func UnpairedEnclaveClient(transport kr.Transport, persister kr.Persister, timeoutsOverride *Timeouts) EnclaveClientI {
	var timeouts = DefaultTimeouts()
	if timeoutsOverride != nil {
		timeouts = *timeoutsOverride
	}
	return &EnclaveClient{
		Transport:                   transport,
		Persister:                   persister,
		Timeouts:                    timeouts,
		requestCallbacksByRequestID: lru.New(128),
		ackedRequestIDs:             lru.New(128),
	}
}

func (client *EnclaveClient) RequestMe(longTimeout bool) (meResponse *kr.MeResponse, err error) {
	meRequest, err := kr.NewRequest()
	if err != nil {
		log.Error(err)
		return
	}
	meRequest.MeRequest = &kr.MeRequest{}
	timeout := client.Timeouts.Me.Fail
	if longTimeout {
		timeout = client.Timeouts.Pair.Fail
	}
	callback, err := client.tryRequest(meRequest, timeout, client.Timeouts.Me.Alert, "Incoming kr me request. Open Kryptonite to continue.")
	if err != nil {
		log.Error(err)
		return
	}
	if callback != nil {
		response := callback.response
		meResponse = response.MeResponse
		if meResponse != nil {
			client.Lock()
			client.cachedMe = &meResponse.Me
			if persistErr := client.Persister.SaveMe(meResponse.Me); persistErr != nil {
				log.Error("persist me error:", persistErr.Error())
			}
			client.Unlock()
		}
		ioutil.WriteFile(filepath.Join(os.Getenv("HOME"), ".ssh", "id_kryptonite.pub"), []byte(meResponse.Me.AuthorizedKeyString()), 0700)
	}
	return
}

func (client *EnclaveClient) RequestSignature(signRequest kr.SignRequest) (signResponse *kr.SignResponse, err error) {
	start := time.Now()
	request, err := kr.NewRequest()
	if err != nil {
		log.Error(err)
		return
	}
	request.SignRequest = &signRequest
	alertText := "Incoming SSH request. Open Kryptonite to continue."
	ps := client.getPairingSecret()
	if ps != nil {
		alertText = "Request from " + ps.DisplayName()
	}
	callback, err := client.tryRequest(request, client.Timeouts.Sign.Fail, client.Timeouts.Sign.Alert, alertText)
	if err != nil {
		if err == ErrTimeout {
			client.postEvent("signature", "timeout", nil, nil)
		} else {
			errStr := err.Error()
			client.postEvent("signature", "error", &errStr, nil)
		}
		log.Error(err)
		return
	}
	if callback != nil {
		response := callback.response
		signResponse = response.SignResponse
		millis := uint64(time.Since(start) / time.Millisecond)
		log.Notice("Signature response took", millis, "ms")
		client.postEvent("signature", "success", &callback.medium, &millis)
		if signResponse.Error != nil {
			log.Error("Signature error:", signResponse.Error)
		}
	}
	return
}

func (client *EnclaveClient) RequestNoOp() (err error) {
	request, err := kr.NewRequest()
	if err != nil {
		log.Error(err)
		return
	}
	requestJson, err := json.Marshal(request)
	if err != nil {
		log.Error(err)
		return
	}
	ps := client.getPairingSecret()
	if ps != nil {
		client.sendMessage(ps, requestJson, false)
	}
	return
}

type callbackT struct {
	response kr.Response
	medium   string
}

func (client *EnclaveClient) tryRequest(request kr.Request, timeout time.Duration, alertTimeout time.Duration, alertText string) (callback *callbackT, err error) {
	if timeout == alertTimeout {
		log.Warning("timeout == alertTimeout, alert may not fire")
	}
	cb := make(chan *callbackT, 5)
	go func() {
		err := client.sendRequestAndReceiveResponses(request, cb, timeout)
		if err != nil {
			log.Error("error sendRequestAndReceiveResponses: ", err.Error())
		}
	}()
	timeoutChan := time.After(timeout)
	sendAlertChan := time.After(alertTimeout)
	func() {
		var ack bool
		for {
			select {
			case callback = <-cb:
				if callback != nil && callback.response.AckResponse != nil {
					ack = true
					log.Notice("request", callback.response.RequestID, "ACKed")
					timeoutChan = time.After(client.Timeouts.ACKDelay)
					break
				}
				return
			case <-timeoutChan:
				err = ErrTimeout
				return
			case <-sendAlertChan:
				if ack {
					break
				}
				client.Lock()
				ps := client.pairingSecret
				client.Unlock()
				requestJson, err := json.Marshal(request)
				if err != nil {
					err = &ProtoError{err}
					continue
				}
				if ps != nil {
					client.Transport.PushAlert(ps, alertText, requestJson)
				}
			}
		}
	}()
	if callback == nil && !client.IsPaired() {
		err = ErrNotPaired
	}
	return
}

//	Send one request and receive pending responses, not necessarily associated
//	with this request
func (client *EnclaveClient) sendRequestAndReceiveResponses(request kr.Request, cb chan *callbackT, timeout time.Duration) (err error) {
	pairingSecret := client.getPairingSecret()
	if pairingSecret == nil {
		err = errors.New("EnclaveClient pairing never initiated")
		return
	}
	requestJson, err := json.Marshal(request)
	if err != nil {
		err = &ProtoError{err}
		return
	}

	timeoutAt := time.Now().Add(timeout)

	client.Lock()
	client.requestCallbacksByRequestID.Add(request.RequestID, cb)
	client.Unlock()

	err = client.sendMessage(pairingSecret, requestJson, true)

	if err != nil {
		switch err.(type) {
		case *SendQueued:
			log.Notice(err)
			err = nil
		default:
			return
		}
	}

	receive := func() (numReceived int, err error) {
		ciphertexts, err := client.Transport.Read(pairingSecret)
		if err != nil {
			err = &RecvError{err}
			return
		}

		for _, ctxt := range ciphertexts {
			ctxtErr := client.handleCiphertext(ctxt, "sqs")
			switch ctxtErr {
			case kr.ErrWaitingForKey:
			default:
				err = ctxtErr
			}
		}
		return
	}

	for {
		n, err := receive()
		client.Lock()
		_, requestPending := client.requestCallbacksByRequestID.Get(request.RequestID)
		_, requestAcked := client.ackedRequestIDs.Get(request.RequestID)
		client.Unlock()
		timeout := timeoutAt
		if requestAcked {
			timeout = timeout.Add(client.Timeouts.ACKDelay)
		}
		if err != nil || (n == 0 && time.Now().After(timeout)) || !requestPending {
			if err != nil {
				log.Error("queue err:", err)
			}
			break
		}
	}
	client.Lock()
	if cb, ok := client.requestCallbacksByRequestID.Get(request.RequestID); ok {
		//	request still not processed, give up on it
		cb.(chan *callbackT) <- nil
		client.requestCallbacksByRequestID.Remove(request.RequestID)
		log.Error("evicting request", request.RequestID)
	}
	client.Unlock()

	return
}

func (client *EnclaveClient) handleCiphertext(ciphertext []byte, medium string) (err error) {
	pairingSecret := client.getPairingSecret()
	if pairingSecret == nil {
		err = errors.New("EnclaveClient pairing never initiated")
		return
	}
	unwrappedCiphertext, didUnwrapKey, err := pairingSecret.UnwrapKeyIfPresent(ciphertext)
	if err != nil {
		err = &ProtoError{err}
		return
	}
	if didUnwrapKey {
		client.Lock()
		queue := client.outgoingQueue
		client.outgoingQueue = [][]byte{}
		client.Unlock()

		savePairingErr := client.Persister.SavePairing(pairingSecret)
		if savePairingErr != nil {
			log.Error("error saving pairing:", savePairingErr.Error())
		}

		for _, queuedMessage := range queue {
			err = client.sendMessage(pairingSecret, queuedMessage, true)
			if err != nil {
				log.Error("error sending queued message:", err.Error())
			}
		}
	}
	if unwrappedCiphertext == nil {
		return
	}
	message, err := pairingSecret.DecryptMessage(*unwrappedCiphertext)
	if err != nil {
		log.Error("decrypt error:", err)
		return
	}
	if message == nil {
		return
	}
	responseJson := *message
	err = client.handleMessage(pairingSecret, responseJson, medium)
	if err != nil {
		log.Error("handleMessage error:", err)
		return
	}
	return
}

func (client *EnclaveClient) sendMessage(pairingSecret *kr.PairingSecret, message []byte, queue bool) (err error) {
	ciphertext, err := pairingSecret.EncryptMessage(message)
	if err != nil {
		if err == kr.ErrWaitingForKey {
			client.Lock()
			if len(client.outgoingQueue) < 128 && queue {
				client.outgoingQueue = append(client.outgoingQueue, message)
			}
			client.Unlock()
			err = &SendQueued{err}
		} else {
			err = &SendError{err}
		}
		return
	}
	go func() {
		if client.bt == nil {
			return
		}
		err := client.bt.Write(ciphertext)
		if err != nil {
			log.Error("error writing to Bluetooth", err)
		}
	}()

	err = client.Transport.SendMessage(pairingSecret, message)
	if err != nil {
		err = &SendError{err}
		return
	}
	return
}

func (client *EnclaveClient) handleMessage(fromPairing *kr.PairingSecret, message []byte, medium string) (err error) {
	var response kr.Response
	err = json.Unmarshal(message, &response)
	if err != nil {
		return
	}
	client.Lock()
	defer client.Unlock()

	if response.UnpairResponse != nil {
		log.Notice("Received unpair command from phone.")
		client.unpair(fromPairing, false)
		//	cancel all pending callbacks
		client.requestCallbacksByRequestID.OnEvicted = func(key lru.Key, callback interface{}) {
			callback.(chan *callbackT) <- nil
		}
		for client.requestCallbacksByRequestID.Len() > 0 {
			client.requestCallbacksByRequestID.RemoveOldest()
		}
		client.requestCallbacksByRequestID.OnEvicted = nil
		return
	}

	if client.pairingSecret != nil && client.pairingSecret.Equals(fromPairing) {
		if response.SNSEndpointARN != nil {
			client.pairingSecret.SetSNSEndpointARN(response.SNSEndpointARN)
			client.Persister.SavePairing(client.pairingSecret)
		}
		if response.ApprovedUntil != client.pairingSecret.ApprovedUntil {
			client.pairingSecret.ApprovedUntil = response.ApprovedUntil
			client.Persister.SavePairing(client.pairingSecret)
		}

		oldTID := client.pairingSecret.GetTrackingID()
		if response.TrackingID != nil && (oldTID == nil || *response.TrackingID != *oldTID) {
			client.pairingSecret.SetTrackingID(response.TrackingID)
			client.Persister.SavePairing(client.pairingSecret)
		}
	}

	if requestCb, ok := client.requestCallbacksByRequestID.Get(response.RequestID); ok {
		log.Info("found callback for request", response.RequestID)
		requestCb.(chan *callbackT) <- &callbackT{
			response: response,
			medium:   medium,
		}
	} else {
		log.Info("callback not found for request", response.RequestID)
	}
	if response.AckResponse != nil {
		client.ackedRequestIDs.Add(response.RequestID, nil)
	} else {
		client.requestCallbacksByRequestID.Remove(response.RequestID)
	}
	return
}
