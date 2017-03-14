package kr

import (
	"fmt"
	"runtime/debug"

	"github.com/op/go-logging"
)

func RecoverToLog(f func(), log *logging.Logger) {
	defer func() {
		if x := recover(); x != nil {
			if log != nil {
				log.Error(fmt.Sprintf("run time panic: %v", x))
				log.Error(string(debug.Stack()))
			}
		}
	}()
	f()
}
