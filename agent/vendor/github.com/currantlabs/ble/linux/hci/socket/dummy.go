// +build !linux

package socket

import "io"

// NewSocket is a dummy function for non-Linux platform.
func NewSocket(id int) (io.ReadWriteCloser, error) {
	return nil, nil
}
