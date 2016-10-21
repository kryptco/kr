package krdclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/agrinman/kr"
)

var ErrNotPaired = fmt.Errorf("Workstation not yet paired. Please run \"kr pair\" and scan the QRCode with the Kryptonite mobile app.")
var ErrTimedOut = fmt.Errorf("Request timed out. Make sure your phone and workstation are paired and connected to the internet and the Kryptonite app is running.")
var ErrSigning = fmt.Errorf("Kryptonite was unable to perform SSH login. Please restart the Kryptonite app on your phone.")
var ErrRejected = fmt.Errorf("Rejected by Kryptonite.")

func RequestMe() (me kr.Profile, err error) {
	meRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	meRequest.MeRequest = &kr.MeRequest{}

	response, err := makeRequestWithJsonResponse(meRequest)
	if err != nil {
		return
	}

	if response.MeResponse != nil {
		me = response.MeResponse.Me
		return
	}
	err = fmt.Errorf("Response missing profile")
	return
}

func RequestList() (profiles []kr.Profile, err error) {
	meRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	meRequest.ListRequest = &kr.ListRequest{}

	response, err := makeRequestWithJsonResponse(meRequest)
	if err != nil {
		return
	}

	if response.ListResponse != nil {
		profiles = response.ListResponse.Profiles
		return
	}
	err = fmt.Errorf("Response missing profile")
	return
}

func makeRequestWithJsonResponse(request kr.Request) (response kr.Response, err error) {
	daemonConn, err := kr.DaemonDial()
	if err != nil {
		return
	}

	httpRequest, err := request.HTTPRequest()
	if err != nil {
		return
	}
	err = httpRequest.Write(daemonConn)
	if err != nil {
		return
	}

	responseReader := bufio.NewReader(daemonConn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode == http.StatusNotFound {
		err = ErrNotPaired
		return
	}
	if httpResponse.StatusCode == http.StatusInternalServerError {
		err = ErrTimedOut
		return
	}
	if httpResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("Error %d", httpResponse.StatusCode)
		return
	}

	err = json.NewDecoder(httpResponse.Body).Decode(&response)
	if err != nil {
		return
	}
	return
}

func Sign(pkFingerprint []byte, data []byte) (signature []byte, err error) {
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
		Command:              currentCommand(),
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
	if signResponse := krResponse.SignResponse; signResponse != nil {
		if signResponse.Signature != nil {
			signature = *signResponse.Signature
			return
		}
		if signResponse.Error != nil {
			if *signResponse.Error == "rejected" {
				err = ErrRejected
			} else {
				err = ErrSigning
			}
			return
		}
	}
	err = fmt.Errorf("response missing signature")
	return
}
