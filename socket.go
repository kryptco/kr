package kr

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

func NotifyDirFile(file string) (fullPath string, err error) {
	notifyPath, err := NotifyDir()
	if err != nil {
		return
	}
	fullPath = filepath.Join(notifyPath, file)
	return
}

func KrDirFile(file string) (fullPath string, err error) {
	krPath, err := KrDir()
	if err != nil {
		return
	}
	fullPath = filepath.Join(krPath, file)
	return
}

func pingDaemon() (err error) {
	conn, err := DaemonDial()
	if err != nil {
		return
	}
	defer conn.Close()

	pingRequest, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		return
	}
	err = pingRequest.Write(conn)
	if err != nil {
		return
	}
	responseReader := bufio.NewReader(conn)
	_, err = http.ReadResponse(responseReader, pingRequest)
	if err != nil {
		err = fmt.Errorf("Daemon Read error: %s", err.Error())
		return
	}
	return
}

func DaemonDialWithTimeout() (conn net.Conn, err error) {
	done := make(chan error, 1)
	go func() {
		done <- pingDaemon()
	}()

	select {
	case <-time.After(time.Second):
		err = fmt.Errorf("ping timed out")
		return
	case err = <-done:
	}
	if err != nil {
		return
	}

	conn, err = DaemonDial()
	return
}
