package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/agrinman/krssh"
)

func getMe() (me krssh.Profile, err error) {
	daemonConn, err := krssh.DaemonDial()
	if err != nil {
		//	TODO: restart daemon?
		err = fmt.Errorf("DaemonDial error: %s", err.Error())
		return
	}

	meRequest, err := krssh.NewRequest()
	if err != nil {
		return
	}
	meRequest.MeRequest = &krssh.MeRequest{}

	httpRequest, err := meRequest.HTTPRequest()
	if err != nil {
		return
	}
	err = httpRequest.Write(daemonConn)
	if err != nil {
		err = fmt.Errorf("Daemon Write error: %s", err.Error())
		return
	}

	responseReader := bufio.NewReader(daemonConn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode == http.StatusNotFound {
		krssh.DesktopNotify("Not paired, please run \"kr pair\" and scan the QR code with kryptonite.")
	}
	if httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("Non-200 status code %d", httpResponse.StatusCode)
		return
	}

	var krResponse krssh.Response
	err = json.NewDecoder(httpResponse.Body).Decode(&krResponse)
	if err != nil {
		err = fmt.Errorf("Daemon decode error: %s", err.Error())
		return
	}
	if krResponse.MeResponse != nil {
		me = krResponse.MeResponse.Me
		return
	}
	err = fmt.Errorf("response missing profile")
	return
}

func sign(pkFingerprint []byte, data []byte) (signature []byte, err error) {
	daemonConn, err := krssh.DaemonDial()
	if err != nil {
		//	TODO: restart daemon?
		err = fmt.Errorf("DaemonDial error: %s", err.Error())
		return
	}

	signRequest, err := krssh.NewRequest()
	if err != nil {
		return
	}
	signRequest.SignRequest = &krssh.SignRequest{
		PublicKeyFingerprint: pkFingerprint,
		Digest:               data,
	}

	httpRequest, err := signRequest.HTTPRequest()
	if err != nil {
		return
	}
	err = httpRequest.Write(daemonConn)
	if err != nil {
		err = fmt.Errorf("Daemon Write error: %s", err.Error())
		return
	}

	responseReader := bufio.NewReader(daemonConn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("Non-200 status code %d", httpResponse.StatusCode)
		return
	}

	var krResponse krssh.Response
	err = json.NewDecoder(httpResponse.Body).Decode(&krResponse)
	if err != nil {
		err = fmt.Errorf("Daemon decode error: %s", err.Error())
		return
	}
	if signResponse := krResponse.SignResponse; signResponse != nil {
		if signResponse.Signature != nil {
			signature = *signResponse.Signature
			return
		}
		//	TODO: handle sign error
		return
	}
	err = fmt.Errorf("response missing signature")
	return
}
