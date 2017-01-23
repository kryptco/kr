package kr

import (
	"fmt"
)

var ErrNotPaired = fmt.Errorf("Workstation not yet paired. Please run \"kr pair\" and scan the QRCode with the Kryptonite mobile app.")
var ErrTimedOut = fmt.Errorf("Request timed out. Make sure your phone and workstation are paired and connected to the internet and the Kryptonite app is running.")
var ErrSigning = fmt.Errorf("Kryptonite was unable to perform SSH login. Please restart the Kryptonite app on your phone.")
var ErrRejected = fmt.Errorf("Request Rejected ✘")
var ErrConnectingToDaemon = fmt.Errorf("Could not connect to Kryptonite daemon. Make sure it is running by typing \"kr restart\".")
