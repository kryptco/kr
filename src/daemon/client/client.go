package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	version2 "krypt.co/kr/common/version"
	"net"
	"net/http"

	"github.com/blang/semver"

	. "krypt.co/kr/common/protocol"
	. "krypt.co/kr/common/socket"
	. "krypt.co/kr/common/util"
)

var ErrOldKrdRunning = fmt.Errorf(Red("An old version of krd is still running. Please run " + Cyan("kr restart") + Red(" and try again.")))

func IsLatestKrdRunning() (isRunning bool, err error) {
	version, err := RequestKrdVersion()
	if err != nil {
		return
	}
	isRunning = version.Compare(version2.CURRENT_VERSION) == 0
	return
}

func RequestKrdVersionOver(conn net.Conn) (version semver.Version, err error) {
	httpRequest, err := http.NewRequest("GET", "/version", nil)
	if err != nil {
		return
	}
	err = httpRequest.Write(conn)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}

	responseReader := bufio.NewReader(conn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		err = ErrConnectingToDaemon
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
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	defer daemonConn.Close()
	version, err = RequestKrdVersionOver(daemonConn)
	return
}

func RequestMeForceRefresh(userID *string) (me Profile, err error) {
	conn, err := DaemonDialWithTimeout(DaemonSocketOrFatal())
	if err != nil {
		return
	}
	defer conn.Close()
	return RequestMeForceRefreshOver(conn, userID)
}

func RequestMeForceRefreshOver(conn net.Conn, userID *string) (me Profile, err error) {
	meRequestJSON, err := json.Marshal(MeRequest{PGPUserId: userID})
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

func RequestMeOver(conn net.Conn) (me Profile, err error) {
	meRequest, err := NewRequest()
	if err != nil {
		return
	}
	meRequest.MeRequest = &MeRequest{}

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

func RequestMe() (me Profile, err error) {
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	defer daemonConn.Close()
	me, err = RequestMeOver(daemonConn)
	return
}

func RequestGitSignatureOver(request Request, conn net.Conn) (response Response, err error) {
	response, err = makeRequestWithJsonResponse(conn, request)
	if err != nil {
		return
	}

	if response.GitSignResponse == nil {
		err = fmt.Errorf("Response missing GitSignResponse")
		return
	}
	return
}

func RequestGitSignature(request Request) (response Response, err error) {
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	defer daemonConn.Close()
	response, err = RequestGitSignatureOver(request, daemonConn)
	return
}

func RequestHosts() (response HostsResponse, err error) {
	request, err := NewRequest()
	if err != nil {
		return
	}
	StartControlServerLogger(request.NotifyPrefix())
	request.HostsRequest = &HostsRequest{}

	genericResponse, err := MakeRequest(request)
	if err != nil {
		return
	}
	if genericResponse.HostsResponse == nil {
		err = fmt.Errorf("no HostsResponse found")
		return
	}
	response = *genericResponse.HostsResponse
	return
}

func MakeRequest(request Request) (response Response, err error) {
	latestRunning, err := IsLatestKrdRunning()
	if err != nil {
		return
	}
	if !latestRunning {
		err = ErrOldKrdRunning
		return
	}
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	defer daemonConn.Close()
	response, err = makeRequestWithJsonResponse(daemonConn, request)
	return
}

func makeRequestWithJsonResponse(conn net.Conn, request Request) (response Response, err error) {
	httpRequest, err := request.HTTPRequest()
	if err != nil {
		return
	}
	defer httpRequest.Body.Close()
	err = httpRequest.Write(conn)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}

	responseReader := bufio.NewReader(conn)
	httpResponse, err := http.ReadResponse(responseReader, httpRequest)
	if err != nil {
		err = ErrConnectingToDaemon
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

func signOver(conn net.Conn, pkFingerprint []byte, data []byte) (signature []byte, err error) {
	signRequest, err := NewRequest()
	if err != nil {
		return
	}
	signRequest.SignRequest = &SignRequest{
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

	var krResponse Response
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

func Sign(pkFingerprint []byte, data []byte) (signature []byte, err error) {
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = fmt.Errorf("DaemonDialWithTimeout error: %s", err.Error())
		return
	}
	defer daemonConn.Close()
	return signOver(daemonConn, pkFingerprint, data)
}

func requestNoOpOver(conn net.Conn) (err error) {
	noOpRequest, err := NewRequest()
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
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = fmt.Errorf("DaemonDialWithTimeout error: %s", err.Error())
		return
	}
	defer daemonConn.Close()
	return requestNoOpOver(daemonConn)
}

func requestDashboardOver(conn net.Conn) (err error) {
	getDashboard, err := http.NewRequest("GET", "/dashboard", nil)
	if err != nil {
		return
	}
	err = getDashboard.Write(conn)
	if err != nil {
		err = fmt.Errorf("Daemon Write error: %s", err.Error())
		return
	}

	responseReader := bufio.NewReader(conn)
	httpResponse, err := http.ReadResponse(responseReader, getDashboard)
	if err != nil {
		err = ErrConnectingToDaemon
		return
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		err = ErrConnectingToDaemon
		return
	}

	return
}

func RequestDashboard() (err error) {
	unixFile, err := KrDirFile(DAEMON_SOCKET_FILENAME)
	if err != nil {
		return
	}
	daemonConn, err := DaemonDialWithTimeout(unixFile)
	if err != nil {
		err = fmt.Errorf("DaemonDialWithTimeout error: %s", err.Error())
		return
	}
	defer daemonConn.Close()
	return requestDashboardOver(daemonConn)
}
