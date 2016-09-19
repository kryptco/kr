package main

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"bitbucket.org/kryptco/go.corebluetooth"
	"bitbucket.org/kryptco/krssh"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/groupcache/lru"
	"golang.org/x/crypto/ssh"
	"log"
	"sync"
	"time"
)

var ErrTimeout = errors.New("Request timed out")

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
	Pair(krssh.PairingSecret) (err error)
	RequestMe() (*krssh.MeResponse, error)
	RequestMeSigner() (ssh.Signer, error)
	GetCachedMe() *krssh.Profile
	GetCachedMeSigner() ssh.Signer
	RequestSignature(krssh.SignRequest) (*krssh.SignResponse, error)
	RequestList(krssh.ListRequest) (*krssh.ListResponse, error)
}

type EnclaveClient struct {
	mutex                       sync.Mutex
	pairingSecret               *krssh.PairingSecret
	requestCallbacksByRequestID *lru.Cache
	snsEndpointARN              *string
	cachedMe                    *krssh.Profile
	bt                          *corebluetooth.CoreBluetoothDriver
}

func (ec *EnclaveClient) Pair(pairingSecret krssh.PairingSecret) (err error) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	if ec.pairingSecret != nil && ec.bt != nil {
		oldBTUUID, err := ec.pairingSecret.DeriveBluetoothServiceUUID()
		if err == nil {
			ec.bt.RemoveService(oldBTUUID.String())
		}
	}
	ec.pairingSecret = &pairingSecret
	btUUID, err := ec.pairingSecret.DeriveBluetoothServiceUUID()
	if err != nil {
		return
	}
	if ec.bt == nil {
		ec.bt, err = corebluetooth.NewContextAndDriver()
		if err != nil {
			log.Println(err)
			return
		}
	}
	//	spawn reader
	go func() {
		for ciphertext := range ec.bt.Read {
			ec.mutex.Lock()
			responseJson, err := ec.pairingSecret.DecryptMessage(ciphertext)
			ec.mutex.Unlock()
			if err != nil {
				log.Println("bluetooth decrypt error:", err)
				return
			}
			var response krssh.Response
			err = json.Unmarshal(responseJson, &response)
			if err != nil {
				log.Println("bluetooth response unmarshal error :", err)
				return
			}
			ec.mutex.Lock()
			if cb, ok := ec.requestCallbacksByRequestID.Get(response.RequestID); ok {
				cb.(chan *krssh.Response) <- &response
				ec.requestCallbacksByRequestID.Remove(response.RequestID)
				log.Println("found BT response cb for request", response.RequestID)
			} else {
				log.Println("no BT response cb for request", response.RequestID)
			}
			ec.mutex.Unlock()
		}
	}()
	err = ec.bt.AddService(btUUID.String(), map[string][]byte{krsshCharUUIDString: []byte{1, 0}})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func (ec *EnclaveClient) getPairingSecret() (pairingSecret *krssh.PairingSecret) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	pairingSecret = ec.pairingSecret
	return
}

func (ec *EnclaveClient) GetCachedMe() (me *krssh.Profile) {
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

func (ec *EnclaveClient) proxyKey(me krssh.Profile) (signer ssh.Signer, err error) {
	proxiedKey, err := PKDERToProxiedKey(ec, me.PublicKeyDER)
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

func (client *EnclaveClient) RequestMe() (meResponse *krssh.MeResponse, err error) {
	meRequest, err := krssh.NewRequest()
	if err != nil {
		log.Println(err)
		return
	}
	meRequest.MeRequest = &krssh.MeRequest{}
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
func (client *EnclaveClient) RequestSignature(signRequest krssh.SignRequest) (signResponse *krssh.SignResponse, err error) {
	request, err := krssh.NewRequest()
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
	}
	return
}
func (client *EnclaveClient) RequestList(listRequest krssh.ListRequest) (listResponse *krssh.ListResponse, err error) {
	request, err := krssh.NewRequest()
	if err != nil {
		log.Println(err)
		return
	}
	request.ListRequest = &listRequest
	response, err := client.tryRequest(request, 0)
	if err != nil {
		log.Println(err)
		return
	}
	if response != nil {
		listResponse = response.ListResponse
	} else {
		//	TODO: handle timeout
	}
	return
}

func (client *EnclaveClient) tryRequest(request krssh.Request, timeout time.Duration) (response *krssh.Response, err error) {
	cb := make(chan *krssh.Response, 1)
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
func (client *EnclaveClient) sendRequestAndReceiveResponses(request krssh.Request, cb chan *krssh.Response, timeout time.Duration) (err error) {
	pairingSecret := client.getPairingSecret()
	if pairingSecret == nil {
		err = errors.New("EnclaveClient not paired")
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

	client.mutex.Lock()
	snsEndpointARN := client.snsEndpointARN
	client.mutex.Unlock()
	ciphertext, err := pairingSecret.EncryptMessage(requestJson)
	if err != nil {
		err = &SendError{err}
		return
	}
	go func() {
		ctxtString := base64.StdEncoding.EncodeToString(ciphertext)
		if snsEndpointARN != nil {
			if pushErr := krssh.PushToSNSEndpoint(ctxtString, *snsEndpointARN, pairingSecret.SQSSendQueueName()); pushErr != nil {
				log.Println("Push error:", pushErr)
			}
		}
	}()
	go func() {
		log.Println("writing to peripheral...")
		err := client.bt.WriteData(ciphertext)
		if err != nil {
			log.Println("error writing BT", err)
		}
	}()

	err = pairingSecret.SendMessage(requestJson)
	if err != nil {
		err = &SendError{err}
		return
	}

	receive := func() (numReceived int, err error) {
		responseJsons, err := pairingSecret.ReceiveMessages()
		if err != nil {
			err = &RecvError{err}
			return
		}

		for _, responseJson := range responseJsons {
			var response krssh.Response
			err := json.Unmarshal(responseJson, &response)
			if err != nil {
				continue
			}

			numReceived++

			if response.SNSEndpointARN != nil {
				client.mutex.Lock()
				client.snsEndpointARN = response.SNSEndpointARN
				client.mutex.Unlock()
			}

			client.mutex.Lock()
			if requestCb, ok := client.requestCallbacksByRequestID.Get(response.RequestID); ok {
				log.Println("found callback for request", response.RequestID)
				requestCb.(chan *krssh.Response) <- &response
			} else {
				log.Println("callback not found for request", response.RequestID)
			}
			client.requestCallbacksByRequestID.Remove(response.RequestID)
			client.mutex.Unlock()
		}
		return
	}

	for {
		n, err := receive()
		_, requestPending := client.requestCallbacksByRequestID.Get(request.RequestID)
		if err != nil || (n == 0 && time.Now().After(timeoutAt)) || !requestPending {
			break
		}
	}
	client.mutex.Lock()
	if cb, ok := client.requestCallbacksByRequestID.Get(request.RequestID); ok {
		//	request still not processed, give up on it
		cb.(chan *krssh.Response) <- nil
		client.requestCallbacksByRequestID.Remove(request.RequestID)
		log.Println("evicting request", request.RequestID)
	}
	client.mutex.Unlock()

	return
}
