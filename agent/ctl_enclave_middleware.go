package main

import (
	"bitbucket.org/kryptco/krssh"
	"encoding/json"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
)

//	Handles pairing requests and delegates other requests through enclave
//	client
type CtlEnclaveMiddleware struct {
	enclaveClient EnclaveClientI
	mutex         sync.Mutex
}

func NewCtlEnclaveMiddleware() *CtlEnclaveMiddleware {
	return &CtlEnclaveMiddleware{}
}

func (server *CtlEnclaveMiddleware) handleCtl(listener net.Listener) (err error) {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/pair", server.handlePair)
	httpMux.HandleFunc("/phone", server.handlePhone)
	err = http.Serve(listener, httpMux)
	return
}

func (server *CtlEnclaveMiddleware) handlePair(w http.ResponseWriter, r *http.Request) {
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
	server.mutex.Unlock()
	meResponse, err := server.enclaveClient.RequestMe()
	if err != nil {
		log.Println(err)
		return
	}
	if meResponse != nil {
		proxiedKey, err := PKDERToProxiedKey(server.enclaveClient, meResponse.Me.PublicKeyDER)
		if err != nil {
			log.Println(err)
			return
		}
		signer, err := ssh.NewSignerFromSigner(proxiedKey)
		if err != nil {
			log.Println(err)
			return
		}
		signers = append(signers, signer)
	} else {
		log.Println(err)
		return
	}

	//<-time.After(time.Second)
}

func (server *CtlEnclaveMiddleware) handlePhone(w http.ResponseWriter, r *http.Request) {
}
