package main

import (
	"bitbucket.org/kryptco/krssh"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
)

type CtlServer struct {
	enclaveClient EnclaveClientI
	mutex         sync.Mutex
}

func NewCtlServer() *CtlServer {
	return &CtlServer{}
}

func (server *CtlServer) handleCtl(listener net.Listener) (err error) {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/pair", server.handlePair)
	httpMux.HandleFunc("/phone", server.handlePhone)
	err = http.Serve(listener, httpMux)
	return
}

func (server *CtlServer) handlePair(w http.ResponseWriter, r *http.Request) {
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
	server.mutex.Lock()
	server.enclaveClient = NewEnclaveClient(pairingSecret)
	_, err = server.enclaveClient.RequestMe()
	if err != nil {
		log.Println(err)
		return
	}
	server.mutex.Unlock()

	//<-time.After(time.Second)
}

func (server *CtlServer) handlePhone(w http.ResponseWriter, r *http.Request) {
}
