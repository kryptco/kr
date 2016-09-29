package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/agrinman/kr"
)

type ControlServer struct {
	enclaveClient EnclaveClientI
}

func NewControlServer() *ControlServer {
	return &ControlServer{UnpairedEnclaveClient()}
}

func (cs *ControlServer) HandleControlHTTP(listener net.Listener) (err error) {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/pair", cs.handlePair)
	httpMux.HandleFunc("/enclave", cs.handleEnclave)
	err = http.Serve(listener, httpMux)
	return
}

func (cs *ControlServer) handlePair(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cs.handleGetPair(w, r)
		return
	case http.MethodPut:
		cs.handlePutPair(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

//	check if pairing completed
func (cs *ControlServer) handleGetPair(w http.ResponseWriter, r *http.Request) {
	meResponse, err := cs.enclaveClient.RequestMe()
	if err == nil && meResponse != nil {
		err = json.NewEncoder(w).Encode(meResponse.Me)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		if err != nil {
			log.Println(err)
		}
		return
	}
}

//	initiate new pairing (clearing any existing)
func (cs *ControlServer) handlePutPair(w http.ResponseWriter, r *http.Request) {
	pairingSecret, err := cs.enclaveClient.Pair()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err)
		return
	}
	err = json.NewEncoder(w).Encode(pairingSecret)
	if err != nil {
		log.Println(err)
		return
	}
}

//	route request to enclave
func (cs *ControlServer) handleEnclave(w http.ResponseWriter, r *http.Request) {
	var enclaveRequest kr.Request
	err := json.NewDecoder(r.Body).Decode(&enclaveRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if enclaveRequest.MeRequest != nil {
		cachedMe := cs.enclaveClient.GetCachedMe()
		if cachedMe != nil {
			response := kr.Response{
				MeResponse: &kr.MeResponse{
					Me: *cachedMe,
				},
			}
			err = json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		return
	}

	if enclaveRequest.SignRequest != nil {
		if enclaveRequest.SignRequest.Command == nil {
			enclaveRequest.SignRequest.Command = getLastCommand()
		}
		signResponse, err := cs.enclaveClient.RequestSignature(*enclaveRequest.SignRequest)
		if err != nil {
			log.Println("signature request error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if signResponse != nil {
			response := kr.Response{
				RequestID:    enclaveRequest.RequestID,
				SignResponse: signResponse,
			}
			err = json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	if enclaveRequest.ListRequest != nil {
		listResponse, err := cs.enclaveClient.RequestList(*enclaveRequest.ListRequest)
		if err != nil {
			log.Println("list request error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if listResponse != nil {
			response := kr.Response{
				RequestID:    enclaveRequest.RequestID,
				ListResponse: listResponse,
			}
			err = json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}
