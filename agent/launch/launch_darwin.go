package launch

/*
*	OSX-specific launchd interaction
 */

/*
#include <stdlib.h>
int launch_activate_socket(const char *name, int **fds, size_t *cnt);
#include <string.h>
char* strerror(int errnum);
*/
import "C"

import (
	"errors"
	"fmt"
	"net"
	"os"
	"unsafe"
)

func OpenAuthAndCtlSockets() (authSocket net.Listener, ctlSocket net.Listener, err error) {
	launchdAuthListeners, err := launch.SocketListeners("AuthListener")
	if err != nil {
		return
	}
	if len(launchdAuthListeners) == 0 {
		err = errors.New("no launchd auth listener found")
		return
	}
	launchdCtlListeners, err := launch.SocketListeners("CtlListener")
	if err != nil {
		return
	}
	if len(launchdCtlListeners) == 0 {
		err = errors.New("no launchd ctl listener found")
		return
	}
	authSocket = launchdAuthListners[0]
	ctlSocket = launchdCtlListeners[0]
	return
}

func SocketFiles(name string) ([]*os.File, error) {
	fds, err := activateSocket(name)
	if err != nil {
		return nil, err
	}

	files := make([]*os.File, 0)
	for _, fd := range fds {
		file := os.NewFile(uintptr(fd), "")
		files = append(files, file)
	}

	return files, nil
}

func SocketListeners(name string) ([]net.Listener, error) {
	files, err := SocketFiles(name)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0)
	for _, file := range files {
		listener, err := net.FileListener(file)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

func activateSocket(name string) ([]int, error) {
	c_name := C.CString(name)
	var c_fds *C.int
	c_cnt := C.size_t(0)

	err := C.launch_activate_socket(c_name, &c_fds, &c_cnt)
	if err != 0 {
		errStr := C.GoString(C.strerror(err))
		return nil, errors.New(fmt.Sprintf("couldn't activate launchd socket %s, Error %s", name, errStr))
	}

	length := int(c_cnt)
	pointer := unsafe.Pointer(c_fds)
	fds := (*[1 << 30]C.int)(pointer)
	result := make([]int, length)

	for i := 0; i < length; i++ {
		result[i] = int(fds[i])
	}

	C.free(pointer)
	return result, nil
}
