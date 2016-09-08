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
type CtlEnclaveMiddlewareI interface {
	EnclaveClientI
	HandleCtl(listener net.Listener) (err error)
	RequestMeSigner() (me ssh.Signer, err error)
}

type CtlEnclaveMiddleware struct {
	EnclaveClientI
	mutex             sync.Mutex
	cachedSigner      ssh.Signer
	newEnclaveClientI func(krssh.PairingSecret) EnclaveClientI
}

func NewCtlEnclaveMiddleware() *CtlEnclaveMiddleware {
	return &CtlEnclaveMiddleware{
		newEnclaveClientI: NewEnclaveClient,
	}
}

func (middleware *CtlEnclaveMiddleware) HandleCtl(listener net.Listener) (err error) {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/pair", middleware.handlePair)
	httpMux.HandleFunc("/phone", middleware.handlePhone)
	err = http.Serve(listener, httpMux)
	return
}

func (middleware *CtlEnclaveMiddleware) handlePair(w http.ResponseWriter, r *http.Request) {
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
	middleware.mutex.Lock()
	middleware.EnclaveClientI = middleware.newEnclaveClientI(pairingSecret)
	middleware.mutex.Unlock()

	//	populate cached Me
	middleware.RequestMeSigner()

	//<-time.After(time.Second)
}

func (middleware *CtlEnclaveMiddleware) RequestMeSigner() (me ssh.Signer, err error) {
	middleware.mutex.Lock()
	defer middleware.mutex.Unlock()
	if middleware.EnclaveClientI == nil {
		return
	}
	if middleware.cachedSigner != nil {
		me = middleware.cachedSigner
		return
	}
	meResponse, err := middleware.RequestMe()
	if err != nil {
		log.Println(err)
		return
	}
	if meResponse != nil {
		proxiedKey, pkErr := PKDERToProxiedKey(middleware, meResponse.Me.PublicKeyDER)
		if pkErr != nil {
			err = pkErr
			log.Println(err)
			return
		}
		signer, signerErr := ssh.NewSignerFromSigner(proxiedKey)
		if signerErr != nil {
			err = signerErr
			log.Println(err)
			return
		}
		middleware.cachedSigner = signer
	} else {
		log.Println(err)
		return
	}
	return
}

func (middleware *CtlEnclaveMiddleware) handlePhone(w http.ResponseWriter, r *http.Request) {
}
