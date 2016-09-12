package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"bitbucket.org/kryptco/krssh"
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
	jsonBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	var pairingSecret krssh.PairingSecret
	err = json.Unmarshal(jsonBody, &pairingSecret)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(pairingSecret)
	cs.enclaveClient.Pair(pairingSecret)

	//	response with profile
	meResponse, err := cs.enclaveClient.RequestMe()
	if err == nil && meResponse != nil {
		err = json.NewEncoder(w).Encode(meResponse.Me)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		log.Println(err)
		return
	}

	//<-time.After(time.Second)
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
