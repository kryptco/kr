package kr

import (
	stdlog "log"
	"log/syslog"
	"os"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("")
var syslogFormat = logging.MustStringFormatter(
	`%{time:15:04:05.000} %{level:.6s} ▶ %{message}`,
)
var stderrFormat = logging.MustStringFormatter(
	`%{color}Kryptonite ▶ %{message}%{color:reset}`,
)

func SetupLogging(prefix string, defaultLogLevel logging.Level, trySyslog bool) *logging.Logger {
	var backend logging.Backend
	if trySyslog {
		var err error
		backend, err = logging.NewSyslogBackendPriority(prefix, syslog.LOG_NOTICE)
		if err == nil {
			logging.SetFormatter(syslogFormat)
			//	direct panic output to syslog as well
			if syslogBackend, ok := backend.(*logging.SyslogBackend); ok {
				stdlog.SetOutput(syslogBackend.Writer)
				stdlog.Println("test")
			}
		} else {
			backend = nil
		}

	}
	if backend == nil {
		backend = logging.NewLogBackend(os.Stderr, prefix, 0)
		logging.SetFormatter(stderrFormat)
	}
	leveled := logging.AddModuleLevel(backend)
	switch os.Getenv("KR_LOG_LEVEL") {
	case "CRITICAL":
		leveled.SetLevel(logging.CRITICAL, prefix)
	case "ERROR":
		leveled.SetLevel(logging.ERROR, prefix)
	case "WARNING":
		leveled.SetLevel(logging.WARNING, prefix)
	case "NOTICE":
		leveled.SetLevel(logging.NOTICE, prefix)
	case "INFO":
		leveled.SetLevel(logging.INFO, prefix)
	case "DEBUG":
		leveled.SetLevel(logging.DEBUG, prefix)
	default:
		leveled.SetLevel(defaultLogLevel, prefix)
	}

	logging.SetBackend(leveled)
	return log
}
