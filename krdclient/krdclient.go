package krdclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/kryptco/kr"
)

func RequestMeOver(conn net.Conn) (me kr.Profile, err error) {
	meRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	meRequest.MeRequest = &kr.MeRequest{}

	response, err := makeRequestWithJsonResponse(conn, meRequest)
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

func RequestMe() (me kr.Profile, err error) {
	unixFile, err := kr.KrDirFile(kr.DAEMON_SOCKET_FILENAME)
	if err != nil {
		err = kr.ErrConnectingToDaemon
		return
	}
	daemonConn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = kr.ErrConnectingToDaemon
		return
	}
	defer daemonConn.Close()
	me, err = RequestMeOver(daemonConn)
	return
}

func RequestGitSignatureOver(request kr.Request, conn net.Conn) (gitSignResponse kr.GitSignResponse, err error) {
	response, err := makeRequestWithJsonResponse(conn, request)
	if err != nil {
		return
	}

	if response.GitSignResponse != nil {
		gitSignResponse = *response.GitSignResponse
		return
	}
	err = fmt.Errorf("Response missing GitSignResponse")
	return
}

func RequestGitSignature(request kr.Request) (response kr.GitSignResponse, err error) {
	unixFile, err := kr.KrDirFile(kr.DAEMON_SOCKET_FILENAME)
	if err != nil {
		err = kr.ErrConnectingToDaemon
		return
	}
	daemonConn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = kr.ErrConnectingToDaemon
		return
	}
	defer daemonConn.Close()
	response, err = RequestGitSignatureOver(request, daemonConn)
	return
}

func makeRequestWithJsonResponse(conn net.Conn, request kr.Request) (response kr.Response, err error) {
	httpRequest, err := request.HTTPRequest()
	if err != nil {
		return
	}
	defer httpRequest.Body.Close()
	err = httpRequest.Write(conn)
	if err != nil {
		err = kr.ErrConnectingToDaemon
		return
	}

	responseReader := bufio.NewReader(conn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		err = kr.ErrConnectingToDaemon
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode == http.StatusNotFound {
		err = kr.ErrNotPaired
		return
	}
	if httpResponse.StatusCode == http.StatusInternalServerError {
		err = kr.ErrTimedOut
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

func signOver(conn net.Conn, pkFingerprint []byte, data []byte) (signature []byte, err error) {
	signRequest, err := kr.NewRequest()
	if err != nil {
		return
	}
	signRequest.SignRequest = &kr.SignRequest{
		PublicKeyFingerprint: pkFingerprint,
		Data:                 data,
	}

	httpRequest, err := signRequest.HTTPRequest()
	if err != nil {
		return
	}
	defer httpRequest.Body.Close()
	err = httpRequest.Write(conn)
	if err != nil {
		err = fmt.Errorf("Daemon Write error: %s", err.Error())
		return
	}

	responseReader := bufio.NewReader(conn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode == http.StatusNotFound {
		err = kr.ErrNotPaired
		return
	}
	if httpResponse.StatusCode == http.StatusInternalServerError {
		err = kr.ErrTimedOut
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
				err = kr.ErrRejected
			} else {
				err = kr.ErrSigning
			}
			return
		}
	}
	err = fmt.Errorf("response missing signature")
	return
}

func Sign(pkFingerprint []byte, data []byte) (signature []byte, err error) {
	unixFile, err := kr.KrDirFile(kr.DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	daemonConn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = fmt.Errorf("DaemonDialWithTimeout error: %s", err.Error())
		return
	}
	defer daemonConn.Close()
	return signOver(daemonConn, pkFingerprint, data)
}

func requestNoOpOver(conn net.Conn) (err error) {
	noOpRequest, err := kr.NewRequest()
	if err != nil {
		return
	}

	httpRequest, err := noOpRequest.HTTPRequest()
	if err != nil {
		return
	}
	defer httpRequest.Body.Close()
	err = httpRequest.Write(conn)
	if err != nil {
		err = fmt.Errorf("Daemon Write error: %s", err.Error())
		return
	}
	return
}

func RequestNoOp() (err error) {
	unixFile, err := kr.KrDirFile(kr.DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	daemonConn, err := kr.DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = fmt.Errorf("DaemonDialWithTimeout error: %s", err.Error())
		return
	}
	defer daemonConn.Close()
	return requestNoOpOver(daemonConn)
}
