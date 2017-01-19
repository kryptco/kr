package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/kryptco/kr"
	"github.com/op/go-logging"
)

func useSyslog() bool {
	env := os.Getenv("KR_LOG_SYSLOG")
	if env != "" {
		return env == "true"
	}
	return true
}

var log *logging.Logger = kr.SetupLogging("krd", logging.INFO, useSyslog())

func main() {
	SetBTLogger(log)
	daemonSocket, err := kr.DaemonListen()
	if err != nil {
		log.Fatal(err)
	}
	defer daemonSocket.Close()

	controlServer, err := NewControlServer()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		controlServer.enclaveClient.Start()
		//err := controlServer.HandleControlHTTP(daemonSocket)
		err := ServeKRAgent(controlServer.enclaveClient, daemonSocket)
		if err != nil {
			log.Error("controlServer return:", err)
		}
	}()

	log.Notice("krd launched and listening on UNIX socket")

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	sig, ok := <-stopSignal
	controlServer.enclaveClient.Stop()
	if ok {
		log.Notice("stopping with signal", sig)
	}
}
