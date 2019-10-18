// +build !darwin

package util

import (
	"os"
)

func MachineName() (name string) {
	name, _ = os.Hostname()
	return
}
