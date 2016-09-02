package main

import (
	"bitbucket.org/kryptco/krssh"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

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
		return
	}

	var pairingSecret krssh.PairingSecret
	err = json.Unmarshal(jsonBody, &pairingSecret)
	if err != nil {
		return
	}
	log.Println(pairingSecret)
}

func handlePhone(w http.ResponseWriter, r *http.Request) {
}
