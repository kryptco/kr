package main

import (
	"log"
	"log/syslog"
	"os"
	"os/signal"
	"syscall"

	"github.com/agrinman/kr"
)

func main() {
	//	redirect stdout > stderr
	syscall.Dup2(2, 1)
	logwriter, e := syslog.New(syslog.LOG_NOTICE, "krd")
	if e == nil {
		log.SetOutput(logwriter)
	}

	daemonSocket, err := kr.DaemonListen()
	if err != nil {
		log.Fatal(err)
	}
	defer daemonSocket.Close()

	controlServer := NewControlServer()
	go func() {
		err := controlServer.HandleControlHTTP(daemonSocket)
		if err != nil {
			log.Println("controlServer return:", err)
		}
	}()

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	sig, ok := <-stopSignal
	if ok {
		log.Println("stopping with signal", sig)
	}
}
