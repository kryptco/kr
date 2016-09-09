package main

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"bitbucket.org/kryptco/krssh"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/groupcache/lru"
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
	RequestMe() (*krssh.MeResponse, error)
	RequestSignature(krssh.SignRequest) (*krssh.SignResponse, error)
	RequestList(krssh.ListRequest) (*krssh.ListResponse, error)
}

type EnclaveClient struct {
	pairingSecret               krssh.PairingSecret
	requestCallbacksByRequestID *lru.Cache
	mutex                       sync.Mutex
	snsEndpointARN              *string
}

func NewEnclaveClient(pairingSecret krssh.PairingSecret) EnclaveClientI {
	return &EnclaveClient{
		pairingSecret:               pairingSecret,
		requestCallbacksByRequestID: lru.New(128),
	}
}

func (client *EnclaveClient) RequestMe() (meResponse *krssh.MeResponse, err error) {
	meRequest, err := krssh.NewRequest()
	if err != nil {
		log.Println(err)
		return
	}
	meRequest.MeRequest = &krssh.MeRequest{}
	response, err := client.tryRequest(meRequest)
	if err != nil {
		log.Println(err)
		return
	}
	if response != nil {
		meResponse = response.MeResponse
	} else {
		//	TODO: handle timeout
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
	response, err := client.tryRequest(request)
	if err != nil {
		log.Println(err)
		return
	}
	if response != nil {
		signResponse = response.SignResponse
	} else {
		//	TODO: handle timeout
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
	response, err := client.tryRequest(request)
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

func (client *EnclaveClient) tryRequest(request krssh.Request) (response *krssh.Response, err error) {
	cb := make(chan *krssh.Response, 1)
	go func() {
		err := client.sendRequestAndReceiveResponses(request, cb)
		if err != nil {
			log.Println("error sendRequestAndReceiveResponses: ", err.Error())
		}
	}()
	select {
	case response = <-cb:
		//	TODO:
	case <-time.After(3 * time.Second):
		err = ErrTimeout
	}
	return
}

//	Send one request and receive pending responses, not necessarily associated
//	with this request
func (client *EnclaveClient) sendRequestAndReceiveResponses(request krssh.Request, cb chan *krssh.Response) (err error) {
	requestJson, err := json.Marshal(request)
	if err != nil {
		err = &ProtoError{err}
		return
	}

	err = client.pairingSecret.SendMessage(requestJson)
	if err != nil {
		err = &SendError{err}
		return
	}

	client.mutex.Lock()
	client.requestCallbacksByRequestID.Add(request.RequestID, cb)
	client.mutex.Unlock()

	client.mutex.Lock()
	snsEndpointARN := client.snsEndpointARN
	client.mutex.Unlock()
	if snsEndpointARN != nil {
		//TODO: send notification to SNS endpoint
	}

	receive := func() (numReceived int, err error) {
		responseJsons, err := client.pairingSecret.ReceiveMessages()
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
		if n == 0 || err != nil {
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
