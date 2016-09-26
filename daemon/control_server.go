package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/agrinman/krssh"
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

//	Generate PairingSecret if not present
//	Remove any existing symmetric key
//	Reply with public fields of PairingSecret
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

//	initiate new pairing
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

func (cs *ControlServer) handleEnclave(w http.ResponseWriter, r *http.Request) {
	var enclaveRequest krssh.Request
	err := json.NewDecoder(r.Body).Decode(&enclaveRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if enclaveRequest.MeRequest != nil {
		cachedMe := cs.enclaveClient.GetCachedMe()
		if cachedMe != nil {
			err = json.NewEncoder(w).Encode(*cachedMe)
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}
}
