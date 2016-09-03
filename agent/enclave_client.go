package main

/*
*	Facillitates communication with a mobile phone SSH key enclave.
 */

import (
	"bitbucket.org/kryptco/krssh"
	"encoding/json"
	"sync"
)

var requestCallbacksByRequestID = map[string]chan krssh.Response{}
var requestCallbacksByRequestIDMutex sync.Mutex

//	Send one request and receive one response, not necessarily the response
//	associated with this request
func SendRequestAndReceiveResponse(ps krssh.PairingSecret, request krssh.Request, cb chan krssh.Response) (err error) {

	requestJson, err := json.Marshal(request)
	if err != nil {
		return
	}

	err = ps.SendMessage(requestJson)
	if err != nil {
		return
	}

	requestCallbacksByRequestIDMutex.Lock()
	requestCallbacksByRequestID[request.RequestID] = cb
	requestCallbacksByRequestIDMutex.Unlock()

	ps.SNSEndpointARNMutex.Lock()
	snsEndpointARN := ps.SNSEndpointARN
	ps.SNSEndpointARNMutex.Unlock()
	if snsEndpointARN != nil {
		//TODO: send notification to SNS endpoint
	}

	responseJson, err := ps.ReceiveMessage()
	if err != nil {
		return
	}

	var response krssh.Response
	err = json.Unmarshal(responseJson, &response)
	if err != nil {
		return
	}

	if response.SNSEndpointARN != nil {
		ps.SNSEndpointARNMutex.Lock()
		ps.SNSEndpointARN = response.SNSEndpointARN
		ps.SNSEndpointARNMutex.Unlock()
	}

	requestCallbacksByRequestIDMutex.Lock()
	if requestCb, ok := requestCallbacksByRequestID[response.RequestID]; ok {
		requestCb <- response
	}
	delete(requestCallbacksByRequestID, response.RequestID)
	requestCallbacksByRequestIDMutex.Unlock()

	return
}
