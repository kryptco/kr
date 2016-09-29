package main

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/agrinman/kr"
	"github.com/golang/groupcache/lru"
	"golang.org/x/crypto/ssh"
	"log"
	"sync"
	"time"
)

var ErrTimeout = errors.New("Request timed out")

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
	RequestMe() (*kr.MeResponse, error)
	RequestMeSigner() (ssh.Signer, error)
	GetCachedMe() *kr.Profile
	GetCachedMeSigner() ssh.Signer
	RequestSignature(kr.SignRequest) (*kr.SignResponse, error)
	RequestList(kr.ListRequest) (*kr.ListResponse, error)
}

type EnclaveClient struct {
	mutex                       sync.Mutex
	pairingSecret               *kr.PairingSecret
	requestCallbacksByRequestID *lru.Cache
	outgoingQueue               [][]byte
	snsEndpointARN              *string
	cachedMe                    *kr.Profile
	bt                          BluetoothDriverI
}

func (ec *EnclaveClient) Pair() (pairingSecret kr.PairingSecret, err error) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.cachedMe = nil

	if ec.bt != nil {
		if ec.pairingSecret != nil {
			oldBtUUID, uuidErr := ec.pairingSecret.DeriveUUID()
			if uuidErr == nil {
				btErr := ec.bt.RemoveService(oldBtUUID)
				if btErr != nil {
					log.Println("error removing bluetooth service:", btErr.Error())
				}
			}
		}
	}

	pairingSecret, err = kr.GeneratePairingSecretAndCreateQueues()
	if err != nil {
		log.Println(err)
		return
	}
	//	erase any existing pairing
	ec.pairingSecret = &pairingSecret

	if ec.bt == nil {
		ec.bt, err = NewBluetoothDriver()
		if err != nil {
			log.Println(err)
			return
		}
		//	spawn at most one reader
		go func() {
			readChan, err := ec.bt.ReadChan()
			if err != nil {
				log.Println("error retrieving bluetooth read channel:", err)
				return
			}
			for ciphertext := range readChan {
				err = ec.handleCiphertext(ciphertext)
			}
		}()
	}
	btUUID, uuidErr := ec.pairingSecret.DeriveUUID()
	if uuidErr != nil {
		err = uuidErr
		log.Println(err)
		return
	}
	err = ec.bt.AddService(btUUID)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func (ec *EnclaveClient) getPairingSecret() (pairingSecret *kr.PairingSecret) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	pairingSecret = ec.pairingSecret
	return
}

func (ec *EnclaveClient) GetCachedMe() (me *kr.Profile) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	me = ec.cachedMe
	return
}

func UnpairedEnclaveClient() EnclaveClientI {
	return &EnclaveClient{
		requestCallbacksByRequestID: lru.New(128),
	}
}

func (ec *EnclaveClient) proxyKey(me kr.Profile) (signer ssh.Signer, err error) {
	proxiedKey, err := ProxySSHWireRSAPublicKey(ec, me.SSHWirePublicKey)
	if err != nil {
		return
	}
	signer, err = ssh.NewSignerFromSigner(proxiedKey)
	if err != nil {
		return
	}
	return
}

func (ec *EnclaveClient) GetCachedMeSigner() (signer ssh.Signer) {
	me := ec.GetCachedMe()
	if me != nil {
		signer, _ = ec.proxyKey(*me)
	}
	return
}

func (ec *EnclaveClient) RequestMeSigner() (signer ssh.Signer, err error) {
	meResponse, err := ec.RequestMe()
	if err != nil {
		return
	}
	if meResponse != nil {
		signer, _ = ec.proxyKey(meResponse.Me)
	}
	return
}

func (client *EnclaveClient) RequestMe() (meResponse *kr.MeResponse, err error) {
	meRequest, err := kr.NewRequest()
	if err != nil {
		log.Println(err)
		return
	}
	meRequest.MeRequest = &kr.MeRequest{}
	response, err := client.tryRequest(meRequest, 20*time.Second)
	if err != nil {
		log.Println(err)
		return
	}
	if response != nil {
		meResponse = response.MeResponse
		if meResponse != nil {
			client.mutex.Lock()
			client.cachedMe = &meResponse.Me
			client.mutex.Unlock()
		}
	}
	return
}
func (client *EnclaveClient) RequestSignature(signRequest kr.SignRequest) (signResponse *kr.SignResponse, err error) {
	start := time.Now()
	request, err := kr.NewRequest()
	if err != nil {
		log.Println(err)
		return
	}
	request.SignRequest = &signRequest
	response, err := client.tryRequest(request, 30*time.Second)
	if err != nil {
		log.Println(err)
		return
	}
	if response != nil {
		signResponse = response.SignResponse
		log.Println("successful signature took", int(time.Since(start)/time.Millisecond), "ms")
	}
	return
}
func (client *EnclaveClient) RequestList(listRequest kr.ListRequest) (listResponse *kr.ListResponse, err error) {
	request, err := kr.NewRequest()
	if err != nil {
		log.Println(err)
		return
	}
	request.ListRequest = &listRequest
	response, err := client.tryRequest(request, 5*time.Second)
	if err != nil {
		log.Println(err)
		return
	}
	if response != nil {
		listResponse = response.ListResponse
	}
	return
}

func (client *EnclaveClient) tryRequest(request kr.Request, timeout time.Duration) (response *kr.Response, err error) {
	cb := make(chan *kr.Response, 1)
	go func() {
		err := client.sendRequestAndReceiveResponses(request, cb, timeout)
		if err != nil {
			log.Println("error sendRequestAndReceiveResponses: ", err.Error())
		}
	}()
	select {
	case response = <-cb:
	case <-time.After(timeout):
		err = ErrTimeout
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

	client.mutex.Lock()
	client.requestCallbacksByRequestID.Add(request.RequestID, cb)
	client.mutex.Unlock()

	err = client.sendMessage(requestJson)

	if err != nil {
		switch err.(type) {
		case *SendQueued:
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
		_, requestPending := client.requestCallbacksByRequestID.Get(request.RequestID)
		if err != nil || (n == 0 && time.Now().After(timeoutAt)) || !requestPending {
			log.Println("done reading queue, err:", err)
			break
		}
	}
	client.mutex.Lock()
	if cb, ok := client.requestCallbacksByRequestID.Get(request.RequestID); ok {
		//	request still not processed, give up on it
		cb.(chan *kr.Response) <- nil
		client.requestCallbacksByRequestID.Remove(request.RequestID)
		log.Println("evicting request", request.RequestID)
	}
	client.mutex.Unlock()

	return
}

func (client *EnclaveClient) handleCiphertext(ciphertext []byte) (err error) {
	unwrappedCiphertext, didUnwrapKey, err := client.pairingSecret.UnwrapKeyIfPresent(ciphertext)
	if err != nil {
		err = &ProtoError{err}
		return
	}
	if didUnwrapKey {
		client.mutex.Lock()
		queue := client.outgoingQueue
		client.outgoingQueue = [][]byte{}
		client.mutex.Unlock()
		for _, queuedMessage := range queue {
			err = client.sendMessage(queuedMessage)
			if err != nil {
				log.Println("error sending queued message:", err.Error())
			}
		}
	}
	if unwrappedCiphertext == nil {
		return
	}
	client.mutex.Lock()
	message, err := client.pairingSecret.DecryptMessage(*unwrappedCiphertext)
	client.mutex.Unlock()
	if err != nil {
		log.Println("decrypt error:", err)
		return
	}
	if message == nil {
		return
	}
	responseJson := *message
	err = client.handleMessage(responseJson)
	if err != nil {
		log.Println("handleMessage error:", err)
		return
	}
	return
}

func (client *EnclaveClient) sendMessage(message []byte) (err error) {
	pairingSecret := client.getPairingSecret()
	if pairingSecret == nil {
		err = errors.New("EnclaveClient pairing never initiated")
		return
	}
	client.mutex.Lock()
	snsEndpointARN := client.snsEndpointARN
	client.mutex.Unlock()
	ciphertext, err := client.pairingSecret.EncryptMessage(message)
	if err != nil {
		if err == kr.ErrWaitingForKey {
			client.mutex.Lock()
			if len(client.outgoingQueue) < 128 {
				client.outgoingQueue = append(client.outgoingQueue, message)
			}
			client.mutex.Unlock()
			err = &SendQueued{err}
		} else {
			err = &SendError{err}
		}
		return
	}
	go func() {
		ctxtString := base64.StdEncoding.EncodeToString(ciphertext)
		if snsEndpointARN != nil {
			if pushErr := kr.PushToSNSEndpoint(ctxtString, *snsEndpointARN, pairingSecret.SQSSendQueueName()); pushErr != nil {
				log.Println("Push error:", pushErr)
			}
		}
	}()
	go func() {
		err := client.bt.Write(ciphertext)
		if err != nil {
			log.Println("error writing BT", err)
		}
	}()

	err = pairingSecret.SendMessage(message)
	if err != nil {
		err = &SendError{err}
		return
	}
	return
}

func (client *EnclaveClient) handleMessage(message []byte) (err error) {
	var response kr.Response
	err = json.Unmarshal(message, &response)
	if err != nil {
		return
	}

	if response.SNSEndpointARN != nil {
		client.mutex.Lock()
		client.snsEndpointARN = response.SNSEndpointARN
		client.mutex.Unlock()
	}

	client.mutex.Lock()
	if requestCb, ok := client.requestCallbacksByRequestID.Get(response.RequestID); ok {
		log.Println("found callback for request", response.RequestID)
		requestCb.(chan *kr.Response) <- &response
	} else {
		log.Println("callback not found for request", response.RequestID)
	}
	client.requestCallbacksByRequestID.Remove(response.RequestID)
	client.mutex.Unlock()
	return
}
