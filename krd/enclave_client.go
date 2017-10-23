package krd

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/golang/groupcache/lru"
	"github.com/kryptco/kr"
	"github.com/op/go-logging"
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
	Pair(kr.PairingOptions) (pairing *kr.PairingSecret, err error)
	IsPaired() bool
	Unpair()
	Start() (err error)
	Stop() (err error)
	RequestMe(meRequest kr.MeRequest, isPairing bool) (*kr.MeResponse, error)
	GetCachedMe() *kr.Profile
	RequestSignature(kr.SignRequest, func()) (*kr.SignResponse, semver.Version, error)
	RequestGitSignature(kr.GitSignRequest, func()) (*kr.GitSignResponse, semver.Version, error)
	RequestGeneric(kr.Request, func()) (kr.Response, error)
	RequestNoOp() error
}

type EnclaveClient struct {
	sync.Mutex
	kr.Transport
	kr.Timeouts
	kr.Persister
	pairingSecret               *kr.PairingSecret
	requestCallbacksByRequestID *lru.Cache
	ackedRequestIDs             *lru.Cache
	outgoingQueue               [][]byte
	snsEndpointARN              *string
	cachedMe                    *kr.Profile
	bt                          BluetoothDriverI
	log                         *logging.Logger
	notifier                    *kr.Notifier
	lastActivityByMedium        map[string]time.Time
}

const BLUETOOTH = "bluetooth"
const SQS = "sqs"

func (ec *EnclaveClient) Pair(pairingOptions kr.PairingOptions) (pairingSecret *kr.PairingSecret, err error) {
	ec.Lock()
	defer ec.Unlock()

	err = ec.generatePairing(pairingOptions)
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

func (ec *EnclaveClient) generatePairing(pairingOptions kr.PairingOptions) (err error) {
	if ec.pairingSecret != nil {
		ec.unpair(ec.pairingSecret, true)
	}
	ec.Persister.DeleteMe()
	ec.Persister.DeletePairing()

	pairingSecret, err := kr.GeneratePairingSecret(pairingOptions.WorkstationName)
	if err != nil {
		ec.log.Error(err)
		return
	}

	go func() {
		setupErr := ec.Transport.Setup(pairingSecret)
		if setupErr != nil {
			ec.log.Error(setupErr)
		}
	}()

	//	erase any existing pairing
	ec.pairingSecret = pairingSecret
	ec.outgoingQueue = [][]byte{}

	savePairingErr := ec.Persister.SavePairing(pairingSecret)
	if savePairingErr != nil {
		ec.log.Error("error saving pairing:", savePairingErr.Error())
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
				ec.log.Error("error creating request:", err)
				return
			}
			unpairRequest.UnpairRequest = &kr.UnpairRequest{}
			unpairJson, err := json.Marshal(unpairRequest)
			if err != nil {
				ec.log.Error("error creating request:", err)
				return
			}
			go ec.sendMessage(pairingSecret, unpairJson, false, false, false)
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
				ec.log.Error("error removing bluetooth service:", btErr.Error())
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
				ec.log.Error(err)
				return
			}
			btErr := ec.bt.AddService(btUUID)
			if btErr != nil {
				ec.log.Error(btErr)
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
		ec.log.Notice("pairing not loaded:", loadErr)
	}

	if loadedMe, loadMeErr := ec.Persister.LoadMe(); loadMeErr == nil {
		ec.cachedMe = &loadedMe
	} else {
		ec.log.Notice("me not loaded:", loadErr)
	}

	bt, err := NewBluetoothDriver()
	if err != nil {
		ec.log.Error("error starting bluetooth driver:", err)
	} else {
		ec.bt = bt
		go func() {
			readChan, err := ec.bt.ReadChan()
			if err != nil {
				ec.log.Error("error retrieving bluetooth read channel:", err)
				return
			}
			for ciphertext := range readChan {
				err = ec.handleCiphertext(ciphertext, BLUETOOTH)
				if err != nil {
					ec.log.Error("error reading bluetooth channel:", err)
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

func UnpairedEnclaveClient(transport kr.Transport, persister kr.Persister, timeoutsOverride *kr.Timeouts, log *logging.Logger, notifier *kr.Notifier) EnclaveClientI {
	var timeouts = kr.DefaultTimeouts()
	if timeoutsOverride != nil {
		timeouts = *timeoutsOverride
	}
	return &EnclaveClient{
		Transport:                   transport,
		Persister:                   persister,
		Timeouts:                    timeouts,
		requestCallbacksByRequestID: lru.New(128),
		ackedRequestIDs:             lru.New(128),
		log:                         log,
		notifier:                    notifier,
		lastActivityByMedium:        map[string]time.Time{},
	}
}

func (client *EnclaveClient) RequestMe(meSubrequest kr.MeRequest, isPairing bool) (meResponse *kr.MeResponse, err error) {
	if !isPairing && !client.IsPaired() {
		err = ErrNotPaired
		return
	}
	meRequest, err := kr.NewRequest()
	if err != nil {
		client.log.Error(err)
		return
	}
	meRequest.MeRequest = &meSubrequest
	if meRequest.MeRequest.PGPUserId == nil {
		client.log.Notice("no PGP user ID in me request, krd invoking git")
		gitUserId, err := kr.GlobalGitUserId()
		if err == nil {
			meRequest.MeRequest.PGPUserId = &gitUserId
		} else {
			client.log.Error("error reading git global user ID: " + err.Error())
			err = nil
		}
	}
	timeout := client.Timeouts.Me.Fail
	if isPairing {
		timeout = client.Timeouts.Pair.Fail
	}
	callback, err := client.tryRequest(meRequest, timeout, client.Timeouts.Me.Alert, "Incoming kr me request. Open Kryptonite to continue.", nil)
	if err != nil {
		client.log.Error(err)
		return
	}
	if callback != nil {
		response := callback.response
		meResponse = response.MeResponse
		if meResponse != nil {
			client.Lock()
			client.cachedMe = &meResponse.Me
			if persistErr := client.Persister.SaveMe(meResponse.Me); persistErr != nil {
				client.log.Error("persist me error:", persistErr.Error())
			}
			client.Persister.SaveMySSHPubKey(meResponse.Me)
			client.Unlock()
		}
	}
	return
}

func (client *EnclaveClient) RequestSignature(signRequest kr.SignRequest, onACK func()) (signResponse *kr.SignResponse, enclaveVersion semver.Version, err error) {
	request, err := kr.NewRequest()
	if err != nil {
		client.log.Error(err)
		return
	}
	request.SignRequest = &signRequest
	response, err := client.RequestGeneric(request, onACK)
	if err != nil {
		return
	}
	signResponse = response.SignResponse
	enclaveVersion = response.Version
	return
}

func (client *EnclaveClient) RequestGitSignature(signRequest kr.GitSignRequest, onACK func()) (signResponse *kr.GitSignResponse, enclaveVersion semver.Version, err error) {
	request, err := kr.NewRequest()
	if err != nil {
		client.log.Error(err)
		return
	}
	request.GitSignRequest = &signRequest
	response, err := client.RequestGeneric(request, onACK)
	if err != nil {
		return
	}
	signResponse = response.GitSignResponse
	enclaveVersion = response.Version
	return
}

func (client *EnclaveClient) RequestGeneric(request kr.Request, onACK func()) (response kr.Response, err error) {
	start := time.Now()
	err = request.Prepare()
	if err != nil {
		return
	}
	alertText := request.RequestParameters(client.Timeouts).AlertText
	ps := client.getPairingSecret()
	if ps != nil {
		alertText = "Request from " + ps.DisplayName()
	}
	timeout := request.RequestParameters(client.Timeouts).Timeout

	callback, err := client.tryRequest(request, timeout.Fail, timeout.Alert, alertText, onACK)
	if err != nil {
		if request.AnalyticsTag() != nil {
			if err == ErrTimeout {
				client.postEvent(*request.AnalyticsTag(), "timeout", nil, nil)
			} else {
				errStr := err.Error()
				client.postEvent(*request.AnalyticsTag(), "error", &errStr, nil)
			}
		}
		client.log.Error(err)
		return
	}
	if callback != nil {
		response = callback.response
		millis := uint64(time.Since(start) / time.Millisecond)
		client.log.Notice("response took", millis, "ms")
		if request.AnalyticsTag() != nil {
			client.postEvent(*request.AnalyticsTag(), "success", &callback.medium, &millis)
		}
		if response.Error() != nil {
			client.log.Error("error:", *response.Error())
		}
	}
	return
}

func (client *EnclaveClient) RequestNoOp() (err error) {
	request, err := kr.NewRequest()
	if err != nil {
		client.log.Error(err)
		return
	}
	requestJson, err := json.Marshal(request)
	if err != nil {
		client.log.Error(err)
		return
	}
	ps := client.getPairingSecret()
	if ps != nil {
		client.sendMessage(ps, requestJson, false, false, client.shouldSendAlertFirst())
	}
	return
}

type callbackT struct {
	response kr.Response
	medium   string
}

func (client *EnclaveClient) tryRequest(request kr.Request, timeout time.Duration, alertTimeout time.Duration, alertText string, onACK func()) (callback *callbackT, err error) {
	if timeout == alertTimeout {
		client.log.Warning("timeout == alertTimeout, alert may not fire")
	}
	cb := make(chan *callbackT, 5)
	pairingSecret := client.getPairingSecret()
	if pairingSecret == nil {
		err = ErrNotPaired
		return
	}
	alertImmediate := client.shouldSendAlertFirst()
	go kr.RecoverToLog(func() {
		err := client.sendRequestAndReceiveResponses(pairingSecret, request, cb, timeout, alertImmediate)
		if err != nil {
			client.log.Error("error sendRequestAndReceiveResponses: ", err.Error())
		}
	}, client.log)
	timeoutChan := time.After(timeout)
	sendAlertChan := time.After(alertTimeout)
	if alertImmediate {
		sendAlertChan = nil
	}
	func() {
		var ack bool
		for {
			select {
			case callback = <-cb:
				if callback != nil && callback.response.AckResponse != nil {
					if onACK != nil {
						onACK()
					}
					onACK = nil
					ack = true
					client.log.Notice("request", callback.response.RequestID, "ACKed")
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
					client.log.Notice("pushing alert for request " + request.RequestID)
					client.Transport.PushAlert(ps, "Kryptonite Request", requestJson)
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
func (client *EnclaveClient) sendRequestAndReceiveResponses(pairingSecret *kr.PairingSecret, request kr.Request, cb chan *callbackT, timeout time.Duration, alertFirst bool) (err error) {
	requestJson, err := json.Marshal(request)
	if err != nil {
		err = &ProtoError{err}
		return
	}

	timeoutAt := time.Now().Add(timeout)

	client.Lock()
	client.requestCallbacksByRequestID.Add(request.RequestID, cb)
	client.Unlock()

	err = client.sendMessage(pairingSecret, requestJson, true, true, alertFirst)

	if err != nil {
		switch err.(type) {
		case *SendQueued, *SendError:
			client.log.Notice(err)
			err = nil
		default:
			return
		}
	}

	receive := func() (numReceived int, err error) {
		ciphertexts, err := client.Transport.Read(client.notifier, pairingSecret)
		if err != nil {
			err = &RecvError{err}
			return
		}

		for _, ctxt := range ciphertexts {
			ctxtErr := client.handleCiphertext(ctxt, SQS)
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
		if (n == 0 && time.Now().After(timeout)) || !requestPending {
			break
		}
		if err != nil {
			client.log.Error("queue err:", err)
			<-time.After(time.Second)
		}
	}
	client.Lock()
	if cb, ok := client.requestCallbacksByRequestID.Get(request.RequestID); ok {
		//	request still not processed, give up on it
		cb.(chan *callbackT) <- nil
		client.requestCallbacksByRequestID.Remove(request.RequestID)
		client.log.Error("evicting request", request.RequestID)
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
		if err == kr.ErrWrappedKeyUnsupported && client.notifier != nil {
			client.notifier.Notify(append([]byte(kr.Red("You are running an old version of the Kryptonite app. Please upgrade Kryptonite on your mobile phone before pairing by visiting get.krypt.co.")), '\r', '\n'))
		}
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
			client.log.Error("error saving pairing:", savePairingErr.Error())
		}

		for _, queuedMessage := range queue {
			err = client.sendMessage(pairingSecret, queuedMessage, true, true, client.shouldSendAlertFirst())
			if err != nil {
				client.log.Error("error sending queued message:", err.Error())
			}
		}
	}
	if unwrappedCiphertext == nil {
		return
	}
	message, err := pairingSecret.DecryptMessage(*unwrappedCiphertext)
	if err != nil {
		client.log.Error("decrypt error:", err)
		return
	}
	if message == nil {
		return
	}
	responseJson := *message
	err = client.handleMessage(pairingSecret, responseJson, medium)
	if err != nil {
		client.log.Error("handleMessage error:", err)
		return
	}
	return
}

func (client *EnclaveClient) shouldSendAlertFirst() bool {
	client.Lock()
	defer client.Unlock()
	if lastSQSActivity, ok := client.lastActivityByMedium[SQS]; ok {
		if lastBluetoothActivity, ok := client.lastActivityByMedium[BLUETOOTH]; ok {
			activityDiff := lastSQSActivity.Sub(lastBluetoothActivity)
			if activityDiff < 0 {
				activityDiff = -activityDiff
			}
			if activityDiff > 5*time.Second {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func (client *EnclaveClient) sendMessage(pairingSecret *kr.PairingSecret, message []byte, queue bool, alertAllowed bool, alertFirst bool) (err error) {
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
			client.log.Error("error writing to Bluetooth", err)
		}
	}()

	if alertFirst && alertAllowed {
		err = client.Transport.PushAlert(pairingSecret, "Kryptonite Request", message)
		if err != nil {
			err = &SendError{err}
			return
		}
	} else {
		err = client.Transport.SendMessage(pairingSecret, message)
		if err != nil {
			err = &SendError{err}
			return
		}
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
	client.lastActivityByMedium[medium] = time.Now()

	if response.UnpairResponse != nil {
		client.log.Notice("Received unpair command from phone.")
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
		client.log.Info("found callback for request", response.RequestID)
		requestCb.(chan *callbackT) <- &callbackT{
			response: response,
			medium:   medium,
		}
	} else {
		client.log.Info("callback not found for request", response.RequestID)
	}
	if response.AckResponse != nil {
		client.ackedRequestIDs.Add(response.RequestID, nil)
	} else {
		client.requestCallbacksByRequestID.Remove(response.RequestID)
	}
	return
}
