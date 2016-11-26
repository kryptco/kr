package main

import (
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"

	"github.com/agrinman/kr"
)

type ControlServer struct {
	enclaveClient EnclaveClientI
}

func NewControlServer() (cs *ControlServer, err error) {
	krdir, err := kr.KrDir()
	if err != nil {
		return
	}
	cs = &ControlServer{UnpairedEnclaveClient(
		kr.AWSTransport{},
		kr.FilePersister{
			PairingDir: krdir,
			SSHDir:     filepath.Join(kr.UnsudoedHomeDir(), ".ssh"),
		},
	)}
	return
}

func (cs *ControlServer) HandleControlHTTP(listener net.Listener) (err error) {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/pair", cs.handlePair)
	httpMux.HandleFunc("/enclave", cs.handleEnclave)
	httpMux.HandleFunc("/ping", cs.handlePing)
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
	case http.MethodDelete:
		cs.handleDeletePair(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (cs *ControlServer) handleDeletePair(w http.ResponseWriter, r *http.Request) {
	cs.enclaveClient.Unpair()
	w.WriteHeader(http.StatusOK)
	return
}

//	check if pairing completed
func (cs *ControlServer) handleGetPair(w http.ResponseWriter, r *http.Request) {
	meResponse, err := cs.enclaveClient.RequestMe(true)
	if err == nil && meResponse != nil {
		err = json.NewEncoder(w).Encode(meResponse.Me)
		if err != nil {
			log.Error(err)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		if err != nil {
			log.Error(err)
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
		log.Error(err)
		return
	}
	err = json.NewEncoder(w).Encode(pairingSecret)
	if err != nil {
		log.Error(err)
		return
	}
}

//	route request to enclave
func (cs *ControlServer) handleEnclave(w http.ResponseWriter, r *http.Request) {
	if !cs.enclaveClient.IsPaired() {
		//	not paired
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var enclaveRequest kr.Request
	err := json.NewDecoder(r.Body).Decode(&enclaveRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if enclaveRequest.MeRequest != nil {
		cs.handleEnclaveMe(w, enclaveRequest)
		return
	}

	if enclaveRequest.SignRequest != nil {
		cs.handleEnclaveSign(w, enclaveRequest)
		return
	}

	if enclaveRequest.ListRequest != nil {
		cs.handleEnclaveList(w, enclaveRequest)
		return
	}

	cs.enclaveClient.RequestNoOp()

	w.WriteHeader(http.StatusOK)
}

func (cs *ControlServer) handleEnclaveMe(w http.ResponseWriter, enclaveRequest kr.Request) {
	var me kr.Profile
	cachedMe := cs.enclaveClient.GetCachedMe()
	if cachedMe != nil {
		me = *cachedMe
	} else {
		meResponse, err := cs.enclaveClient.RequestMe(false)
		if err != nil {
			log.Error("me request error:", err)
			switch err {
			case ErrNotPaired:
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		if meResponse != nil {
			me = meResponse.Me
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}
	response := kr.Response{
		MeResponse: &kr.MeResponse{
			Me: me,
		},
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error(err)
		return
	}
}

func (cs *ControlServer) handleEnclaveList(w http.ResponseWriter, enclaveRequest kr.Request) {
	listResponse, err := cs.enclaveClient.RequestList(*enclaveRequest.ListRequest)
	if err != nil {
		log.Error("list request error:", err)
		switch err {
		case ErrNotPaired:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if listResponse != nil {
		response := kr.Response{
			RequestID:    enclaveRequest.RequestID,
			ListResponse: listResponse,
		}
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error(err)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (cs *ControlServer) handleEnclaveSign(w http.ResponseWriter, enclaveRequest kr.Request) {
	if enclaveRequest.SignRequest.Command == nil {
		enclaveRequest.SignRequest.Command = getLastCommand()
	}
	signResponse, err := cs.enclaveClient.RequestSignature(*enclaveRequest.SignRequest)
	if err != nil {
		log.Error("signature request error:", err)
		switch err {
		case ErrNotPaired:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if signResponse != nil {
		response := kr.Response{
			RequestID:    enclaveRequest.RequestID,
			SignResponse: signResponse,
		}
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error(err)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

}

func (cs *ControlServer) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
