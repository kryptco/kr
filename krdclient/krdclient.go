package krdclient

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/blang/semver"
	"github.com/kryptco/kr"
)

func IsLatestKrdRunning() (isRunning bool, err error) {
	version, err := RequestKrdVersion()
	if err != nil {
		return
	}
	isRunning = version.Compare(kr.CURRENT_VERSION) == 0
	return
}

func RequestKrdVersionOver(conn net.Conn) (version semver.Version, err error) {
	httpRequest, err := http.NewRequest("GET", "/version", nil)
	if err != nil {
		return
	}
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
	if httpResponse.StatusCode != http.StatusOK {
		err = kr.ErrConnectingToDaemon
		return
	}

	defer httpResponse.Body.Close()
	versionBytes, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return
	}

	version, err = semver.Make(string(versionBytes))
	return
}

func RequestKrdVersion() (version semver.Version, err error) {
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
	version, err = RequestKrdVersionOver(daemonConn)
	return
}

func RequestMeForceRefresh(userID *string) (me kr.Profile, err error) {
	conn, err := kr.DaemonDialWithTimeout(kr.DaemonSocketOrFatal())
	if err != nil {
		return
	}
	defer conn.Close()
	return RequestMeForceRefreshOver(conn, userID)
}

func RequestMeForceRefreshOver(conn net.Conn, userID *string) (me kr.Profile, err error) {
	meRequestJSON, err := json.Marshal(kr.MeRequest{userID})
	if err != nil {
		return
	}
	getPair, err := http.NewRequest("GET", "/pair", bytes.NewReader(meRequestJSON))
	if err != nil {
		return
	}
	err = getPair.Write(conn)
	if err != nil {
		return
	}

	getReader := bufio.NewReader(conn)
	getResponse, err := http.ReadResponse(getReader, getPair)

	if err != nil {
		return
	}
	switch getResponse.StatusCode {
	case http.StatusNotFound, http.StatusInternalServerError:
		err = fmt.Errorf("Failed to communicate with phone, ensure your phone and workstation are connected to the internet and try again.")
		return
	case http.StatusOK:
	default:
		err = fmt.Errorf("Failed to communicate with phone, error %d", getResponse.StatusCode)
		return
	}

	defer getResponse.Body.Close()
	err = json.NewDecoder(getResponse.Body).Decode(&me)
	if err != nil {
		return
	}
	return
}

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

func RequestHostsOver(request kr.Request, conn net.Conn) (hostsResponse kr.HostsResponse, err error) {
	response, err := makeRequestWithJsonResponse(conn, request)
	if err != nil {
		return
	}

	if response.HostsResponse != nil {
		hostsResponse = *response.HostsResponse
		return
	}
	err = fmt.Errorf("Response missing HostsResponse")
	return
}

func RequestHosts() (response kr.HostsResponse, err error) {
	request, err := kr.NewRequest()
	if err != nil {
		return
	}
	kr.StartControlServerLogger(request.NotifyPrefix())
	request.HostsRequest = &kr.HostsRequest{}

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
	response, err = RequestHostsOver(request, daemonConn)
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
