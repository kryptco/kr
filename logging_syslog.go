// +build !windows

package kr

import (
	stdlog "log"
	"log/syslog"

	"github.com/op/go-logging"
)

func GetSyslogBackend(prefix string) logging.Backend {
	var backend logging.Backend
	var err error
	backend, err = logging.NewSyslogBackendPriority(prefix, syslog.LOG_NOTICE)
	if err == nil {
		logging.SetFormatter(syslogFormat)
		//	direct panic output to syslog as well
		if syslogBackend, ok := backend.(*logging.SyslogBackend); ok {
			stdlog.SetOutput(syslogBackend.Writer)
		}
	} else {
		backend = nil
	}
	return backend
}
