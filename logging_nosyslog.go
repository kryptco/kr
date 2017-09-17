// +build windows

package kr

import (
	"github.com/op/go-logging"
)

func GetSyslogBackend(prefix string) logging.Backend {
	return nil
}
