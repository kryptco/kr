// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syncql

import (
	"strconv"
	"strings"
)

// SplitError splits an error message into an offset and the remaining (i.e.,
// rhs of offset) message.
// The query error convention is "<module><optional-rpc>[offset]<remaining-message>".
// If err is nil, (0, "") are returned.
func SplitError(err error) (int64, string) {
	if err == nil {
		return 0, ""
	}
	errMsg := err.Error()
	idx1 := strings.Index(errMsg, "[")
	idx2 := strings.Index(errMsg, "]")
	if idx1 == -1 || idx2 == -1 {
		return 0, errMsg
	}
	offsetString := errMsg[idx1+1 : idx2]
	offset, err := strconv.ParseInt(offsetString, 10, 64)
	if err != nil {
		return 0, errMsg
	}
	return offset, errMsg[idx2+1:]
}
