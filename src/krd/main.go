package main

import (
	"fmt"
	log2 "krypt.co/kr/common/log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/op/go-logging"
	"krypt.co/kr/common/socket"
	"krypt.co/kr/daemon"
	"krypt.co/kr/daemon/control"
)

func useSyslog() bool {
	env := os.Getenv("KR_LOG_SYSLOG")
	if env != "" {
		return env == "true"
	}
	return true
}

var log = log2.SetupLogging("krd", logging.INFO, useSyslog())

func main() {

	defer func() {
		if x := recover(); x != nil {
			log.Error(fmt.Sprintf("run time panic: %v", x))
			log.Error(string(debug.Stack()))
			panic(x)
		}
	}()

	err := daemon.UpgradeSSHConfig()
	if err != nil {
		log.Error(err)
		err = nil
	}

	notifier, err := socket.OpenNotifier("")
	if err != nil {
		log.Fatal(err)
	}
	defer notifier.Close()

	socket.StartNotifyCleanup()

	daemonSocket, err := socket.DaemonListen()
	if err != nil {
		log.Fatal(err)
	}
	defer daemonSocket.Close()

	agentSocket, err := socket.AgentListen()
	if err != nil {
		log.Fatal(err)
	}
	defer agentSocket.Close()

	hostAuthSocket, err := socket.HostAuthListen()
	if err != nil {
		log.Fatal(err)
	}
	defer hostAuthSocket.Close()

	controlServer, err := control.NewControlServer(log, &notifier)
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
		err := daemon.ServeKRAgent(controlServer.EnclaveClient(), agentSocket, hostAuthSocket, log)
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
