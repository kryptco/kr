package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/agrinman/kr"
)

var ErrNotPaired = fmt.Errorf("phone not paired with workstation")
var ErrTimedOut = fmt.Errorf("request timed out")

func getMe() (me kr.Profile, err error) {
	daemonConn, err := kr.DaemonDial()
	if err != nil {
		err = fmt.Errorf("DaemonDial error: %s", err.Error())
		return
	}

	meRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	meRequest.MeRequest = &kr.MeRequest{}

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
		kr.DesktopNotify("Not paired, please run \"kr pair\" and scan the QR code with kryptonite.")
	}
	if httpResponse.StatusCode == http.StatusNotFound {
		err = ErrNotPaired
		return
	}
	if httpResponse.StatusCode == http.StatusInternalServerError {
		err = ErrTimedOut
		return
	}
	if httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("Non-200 status code %d", httpResponse.StatusCode)
		return
	}

	var krResponse kr.Response
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
	daemonConn, err := kr.DaemonDial()
	if err != nil {
		err = fmt.Errorf("DaemonDial error: %s", err.Error())
		return
	}

	signRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	signRequest.SignRequest = &kr.SignRequest{
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
	if httpResponse.StatusCode == http.StatusNotFound {
		err = ErrNotPaired
		return
	}
	if httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("Non-200 status code %d", httpResponse.StatusCode)
		return
	}

	var krResponse kr.Response
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
	}
	err = fmt.Errorf("response missing signature")
	return
}
