package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krd"
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
	krd.SetBTLogger(log)

	defer func() {
		if x := recover(); x != nil {
			log.Error(fmt.Sprintf("run time panic: %v", x))
			log.Error(string(debug.Stack()))
			panic(x)
		}
	}()

	notifier, err := kr.OpenNotifier("")
	if err != nil {
		log.Fatal(err)
	}
	defer notifier.Close()

	kr.StartNotifyCleanup()

	daemonSocket, err := kr.DaemonListen()
	if err != nil {
		log.Fatal(err)
	}
	defer daemonSocket.Close()

	agentSocket, err := kr.AgentListen()
	if err != nil {
		log.Fatal(err)
	}
	defer agentSocket.Close()

	hostAuthSocket, err := kr.HostAuthListen()
	if err != nil {
		log.Fatal(err)
	}
	defer hostAuthSocket.Close()

	controlServer, err := krd.NewControlServer(log, &notifier)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		controlServer.Start()
		err := controlServer.HandleControlHTTP(daemonSocket)
		if err != nil {
			log.Error("controlServer return:", err)
		}
	}()

	go func() {
		err := krd.ServeKRAgent(controlServer.EnclaveClient(), agentSocket, hostAuthSocket, log)
		if err != nil {
			log.Error("agent return:", err)
		}
	}()

	log.Notice("krd launched and listening on UNIX socket")

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	sig, ok := <-stopSignal
	controlServer.Stop()
	if ok {
		log.Notice("stopping with signal", sig)
	}
}
