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
	GetCachedMeSigner() ssh.Signer
}

type CtlEnclaveMiddleware struct {
	EnclaveClientI
	mutex             sync.Mutex
	cachedSigner      ssh.Signer
	cachedProfile     *krssh.Profile
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
	httpMux.HandleFunc("/enclave", middleware.handleEnclave)
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
	middleware.cachedProfile = nil
	middleware.cachedSigner = nil
	middleware.mutex.Unlock()

	//	populate cached Me
	middleware.RequestMeSigner()

	//	response with profile
	middleware.mutex.Lock()
	me := middleware.cachedProfile
	middleware.mutex.Unlock()
	if me != nil {
		err = json.NewEncoder(w).Encode(*me)
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

func (middleware *CtlEnclaveMiddleware) GetCachedMeSigner() (me ssh.Signer) {
	middleware.mutex.Lock()
	defer middleware.mutex.Unlock()
	me = middleware.cachedSigner
	return
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
		middleware.cachedProfile = &meResponse.Me
	} else {
		log.Println(err)
		return
	}
	return
}

func (middleware *CtlEnclaveMiddleware) handleEnclave(w http.ResponseWriter, r *http.Request) {
	var enclaveRequest krssh.Request
	err := json.NewDecoder(r.Body).Decode(&enclaveRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if enclaveRequest.MeRequest != nil {
		if middleware.cachedProfile != nil {
			err = json.NewEncoder(w).Encode(*middleware.cachedProfile)
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
