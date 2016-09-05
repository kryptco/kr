package main

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"bitbucket.org/kryptco/krssh"
	"encoding/json"
	"github.com/golang/groupcache/lru"
	"log"
	"sync"
	"time"
)

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

func NewEnclaveClient(pairingSecret krssh.PairingSecret) *EnclaveClient {
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
func (client *EnclaveClient) RequestSignature(krssh.SignRequest) (response *krssh.SignResponse, err error) {
	return
}
func (client *EnclaveClient) RequestList(krssh.ListRequest) (response *krssh.ListResponse, err error) {
	return
}

func (client *EnclaveClient) tryRequest(request krssh.Request) (response *krssh.Response, err error) {
	cb := make(chan *krssh.Response, 1)
	go client.sendRequestAndReceiveResponses(request, cb)
	select {
	case response = <-cb:
	case <-time.After(3 * time.Second):
	}
	return
}

//	Send one request and receive pending responses, not necessarily associated
//	with this request
func (client *EnclaveClient) sendRequestAndReceiveResponses(request krssh.Request, cb chan *krssh.Response) (err error) {
	requestJson, err := json.Marshal(request)
	if err != nil {
		return
	}

	err = client.pairingSecret.SendMessage(requestJson)
	if err != nil {
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

	responseJsons, err := client.pairingSecret.ReceiveMessages()
	if err != nil {
		return
	}

	for _, responseJson := range responseJsons {
		var response krssh.Response
		err = json.Unmarshal(responseJson, &response)
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
			requestCb.(chan *krssh.Response) <- &response
		}
		client.requestCallbacksByRequestID.Remove(response.RequestID)
		client.mutex.Unlock()
	}

	return
}
