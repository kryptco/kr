// +build !darwin

package kr

import (
	"os"
)

func MachineName() (name string) {
	name, _ = os.Hostname()
	return
}
