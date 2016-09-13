package gatt

import "fmt"

// Device ...
type Device interface{}

// ErrNotSupport means the function is not available for the platform.
var ErrNotSupport = fmt.Errorf("command not support")
