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

var pairingSecret krssh.PairingSecret
var pairingSecretMutex sync.Mutex

func handleCtl(listener net.Listener) (err error) {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/pair", handlePair)
	httpMux.HandleFunc("/phone", handlePhone)
	err = http.Serve(listener, httpMux)
	return
}

func handlePair(w http.ResponseWriter, r *http.Request) {
	jsonBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	pairingSecretMutex.Lock()
	defer pairingSecretMutex.Unlock()
	err = json.Unmarshal(jsonBody, &pairingSecret)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(pairingSecret)

	msg, err := pairingSecret.ReceiveMessage()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("received message", string(msg))
	//<-time.After(time.Second)
}

func handlePhone(w http.ResponseWriter, r *http.Request) {
}
