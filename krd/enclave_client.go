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
	Pair() (pairing kr.PairingSecret, err error)
	IsPaired() bool
	Start() (err error)
	Stop() (err error)
	RequestMe() (*kr.MeResponse, error)
	GetCachedMe() *kr.Profile
	RequestSignature(kr.SignRequest) (*kr.SignResponse, error)
	RequestList(kr.ListRequest) (*kr.ListResponse, error)
}

type EnclaveClient struct {
	sync.Mutex
	pairingSecret               *kr.PairingSecret
	requestCallbacksByRequestID *lru.Cache
	outgoingQueue               [][]byte
	snsEndpointARN              *string
	cachedMe                    *kr.Profile
	bt                          BluetoothDriverI
}

func (ec *EnclaveClient) Pair() (pairingSecret kr.PairingSecret, err error) {
	ec.Lock()
	defer ec.Unlock()

	ec.generatePairing()
	ec.activatePairing()

	pairingSecret = *ec.pairingSecret

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
		ec.unpair(*ec.pairingSecret, true)
	}
	kr.DeletePairing()

	pairingSecret, err := kr.GeneratePairingSecretAndCreateQueues()
	if err != nil {
		log.Error(err)
		return
	}
	//	erase any existing pairing
	ec.pairingSecret = &pairingSecret
	ec.outgoingQueue = [][]byte{}

	savePairingErr := kr.SavePairing(pairingSecret)
	if savePairingErr != nil {
		log.Error("error saving pairing:", savePairingErr.Error())
	}
	return
}

func (ec *EnclaveClient) unpair(pairingSecret kr.PairingSecret, sendUnpairRequest bool) (err error) {
	if ec.pairingSecret == nil || !ec.pairingSecret.Equals(pairingSecret) {
		return
	}
	ec.deactivatePairing(pairingSecret)
	ec.cachedMe = nil
	ec.pairingSecret = nil
	kr.DeletePairing()
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

func (ec *EnclaveClient) deactivatePairing(pairingSecret kr.PairingSecret) (err error) {
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
	return
}

func (ec *EnclaveClient) Start() (err error) {
	ec.Lock()
	loadedPairing, loadErr := kr.LoadPairing()
	if loadErr == nil && loadedPairing != nil {
		ec.pairingSecret = loadedPairing
	} else {
		log.Notice("pairing not loaded:", loadErr)
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
				err = ec.handleCiphertext(ciphertext)
			}
		}()
	}

	ec.activatePairing()
	ec.Unlock()
	if ec.getPairingSecret() != nil {
		go ec.RequestMe()
	}
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

func UnpairedEnclaveClient() EnclaveClientI {
	return &EnclaveClient{
		requestCallbacksByRequestID: lru.New(128),
	}
}

func (client *EnclaveClient) RequestMe() (meResponse *kr.MeResponse, err error) {
	meRequest, err := kr.NewRequest()
	if err != nil {
		log.Error(err)
		return
	}
	meRequest.MeRequest = &kr.MeRequest{}
	response, err := client.tryRequest(meRequest, 20*time.Second, 5*time.Second, "Incoming SSH request. Open Kryptonite to continue.")
	if err != nil {
		log.Error(err)
		return
	}
	if response != nil {
		meResponse = response.MeResponse
		if meResponse != nil {
			client.Lock()
			client.cachedMe = &meResponse.Me
			client.Unlock()
		}
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
	requestTimeout := 15 * time.Second
	alertTimeout := 5 * time.Second
	alertText := "Incoming SSH request. Open Kryptonite to continue."
	ps := client.getPairingSecret()
	if ps != nil && ps.RequireManualApproval {
		requestTimeout = 20 * time.Second
		alertTimeout = 19 * time.Second
		alertText = "Manual approval enabled but app not running. Open Kryptonite to approve requests."
	}
	response, err := client.tryRequest(request, requestTimeout, alertTimeout, alertText)
	if err != nil {
		log.Error(err)
		return
	}
	if response != nil {
		signResponse = response.SignResponse
		log.Notice("successful signature took", int(time.Since(start)/time.Millisecond), "ms")
	}
	return
}
func (client *EnclaveClient) RequestList(listRequest kr.ListRequest) (listResponse *kr.ListResponse, err error) {
	request, err := kr.NewRequest()
	if err != nil {
		log.Error(err)
		return
	}
	request.ListRequest = &listRequest
	response, err := client.tryRequest(request, 10*time.Second, 5*time.Second, "Incoming SSH request. Open Kryptonite to continue.")
	if err != nil {
		log.Error(err)
		return
	}
	if response != nil {
		listResponse = response.ListResponse
	}
	return
}

func (client *EnclaveClient) tryRequest(request kr.Request, timeout time.Duration, alertTimeout time.Duration, alertText string) (response *kr.Response, err error) {
	cb := make(chan *kr.Response, 1)
	go func() {
		err := client.sendRequestAndReceiveResponses(request, cb, timeout)
		if err != nil {
			log.Error("error sendRequestAndReceiveResponses: ", err.Error())
		}
	}()
	timeoutChan := time.After(timeout)
	sendAlertChan := time.After(alertTimeout)
	func() {
		for {
			select {
			case response = <-cb:
				return
			case <-timeoutChan:
				err = ErrTimeout
				return
			case <-sendAlertChan:
				client.Lock()
				ps := client.pairingSecret
				client.Unlock()
				requestJson, err := json.Marshal(request)
				if err != nil {
					err = &ProtoError{err}
					continue
				}
				if ps != nil {
					ps.PushAlert(alertText, requestJson)
				}
			}
		}
	}()
	if response == nil && !client.IsPaired() {
		err = ErrNotPaired
	}
	return
}

//	Send one request and receive pending responses, not necessarily associated
//	with this request
func (client *EnclaveClient) sendRequestAndReceiveResponses(request kr.Request, cb chan *kr.Response, timeout time.Duration) (err error) {
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

	err = client.sendMessage(*pairingSecret, requestJson, true)

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
		ciphertexts, err := pairingSecret.ReadQueue()
		if err != nil {
			err = &RecvError{err}
			return
		}

		for _, ctxt := range ciphertexts {
			ctxtErr := client.handleCiphertext(ctxt)
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
		client.Unlock()
		if err != nil || (n == 0 && time.Now().After(timeoutAt)) || !requestPending {
			if err != nil {
				log.Error("queue err:", err)
			}
			break
		}
	}
	client.Lock()
	if cb, ok := client.requestCallbacksByRequestID.Get(request.RequestID); ok {
		//	request still not processed, give up on it
		cb.(chan *kr.Response) <- nil
		client.requestCallbacksByRequestID.Remove(request.RequestID)
		log.Error("evicting request", request.RequestID)
	}
	client.Unlock()

	return
}

func (client *EnclaveClient) handleCiphertext(ciphertext []byte) (err error) {
	pairingSecret := client.getPairingSecret()
	unwrappedCiphertext, didUnwrapKey, err := pairingSecret.UnwrapKeyIfPresent(ciphertext)
	if pairingSecret == nil {
		err = errors.New("EnclaveClient pairing never initiated")
		return
	}
	if err != nil {
		err = &ProtoError{err}
		return
	}
	if didUnwrapKey {
		client.Lock()
		queue := client.outgoingQueue
		client.outgoingQueue = [][]byte{}
		client.Unlock()

		savePairingErr := kr.SavePairing(*pairingSecret)
		if savePairingErr != nil {
			log.Error("error saving pairing:", savePairingErr.Error())
		}

		for _, queuedMessage := range queue {
			err = client.sendMessage(*pairingSecret, queuedMessage, true)
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
	err = client.handleMessage(*pairingSecret, responseJson)
	if err != nil {
		log.Error("handleMessage error:", err)
		return
	}
	return
}

func (client *EnclaveClient) sendMessage(pairingSecret kr.PairingSecret, message []byte, queue bool) (err error) {
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
		err := client.bt.Write(ciphertext)
		if err != nil {
			log.Error("error writing BT", err)
		}
	}()

	err = pairingSecret.SendMessage(message)
	if err != nil {
		err = &SendError{err}
		return
	}
	return
}

func (client *EnclaveClient) handleMessage(fromPairing kr.PairingSecret, message []byte) (err error) {
	var response kr.Response
	err = json.Unmarshal(message, &response)
	if err != nil {
		return
	}

	if response.UnpairResponse != nil {
		log.Notice("Received unpair command from phone.")
		client.Lock()
		client.unpair(fromPairing, false)
		//	cancel all pending callbacks
		client.requestCallbacksByRequestID.OnEvicted = func(key lru.Key, callback interface{}) {
			callback.(chan *kr.Response) <- nil
		}
		for client.requestCallbacksByRequestID.Len() > 0 {
			client.requestCallbacksByRequestID.RemoveOldest()
		}
		client.requestCallbacksByRequestID.OnEvicted = nil
		client.Unlock()
		return
	}

	if response.SNSEndpointARN != nil {
		client.Lock()
		if client.pairingSecret != nil {
			client.pairingSecret.SetSNSEndpointARN(response.SNSEndpointARN)
			kr.SavePairing(*client.pairingSecret)
		}
		if response.RequireManualApproval != client.pairingSecret.RequireManualApproval {
			client.pairingSecret.RequireManualApproval = response.RequireManualApproval
			kr.SavePairing(*client.pairingSecret)
		}
		client.Unlock()
	}

	client.Lock()
	if requestCb, ok := client.requestCallbacksByRequestID.Get(response.RequestID); ok {
		log.Info("found callback for request", response.RequestID)
		requestCb.(chan *kr.Response) <- &response
	} else {
		log.Info("callback not found for request", response.RequestID)
	}
	client.requestCallbacksByRequestID.Remove(response.RequestID)
	client.Unlock()
	return
}
