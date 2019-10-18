package util

import (
	"fmt"
)

var ErrNotPaired = fmt.Errorf("Workstation not yet paired. Please run \"kr pair\" and scan the QRCode with the Krypton mobile app.")
var ErrTimedOut = fmt.Errorf("Request timed out. Make sure your phone and workstation are paired and connected to the internet and the Krypton app is running.")
var ErrSigning = fmt.Errorf("Krypton was unable to perform SSH login. Please restart the Krypton app on your phone.")
var ErrRejected = fmt.Errorf("Request Rejected ✘")
var ErrConnectingToDaemon = fmt.Errorf("Could not connect to Krypton daemon. Make sure it is running by typing \"kr restart\".")
