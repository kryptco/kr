package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/agrinman/kr"
	"github.com/op/go-logging"
)

var log *logging.Logger

func main() {
	log = kr.SetupLogging("krd", logging.NOTICE, true)
	daemonSocket, err := kr.DaemonListen()
	if err != nil {
		log.Fatal(err)
	}
	defer daemonSocket.Close()

	controlServer := NewControlServer()
	go func() {
		err := controlServer.HandleControlHTTP(daemonSocket)
		if err != nil {
			log.Error("controlServer return:", err)
		}
	}()

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	sig, ok := <-stopSignal
	if ok {
		log.Notice("stopping with signal", sig)
	}
}
