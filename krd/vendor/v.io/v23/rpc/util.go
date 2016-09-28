// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"v.io/v23/context"
	"v.io/v23/glob"
	"v.io/v23/naming"
)

// NewGlobState returns the GlobState corresponding to obj.  Returns nil if obj
// doesn't implement AllGlobber or ChildrenGlobber.
func NewGlobState(obj interface{}) *GlobState {
	a, ok1 := obj.(AllGlobber)
	c, ok2 := obj.(ChildrenGlobber)
	if ok1 || ok2 {
		return &GlobState{
			AllGlobber:      a,
			ChildrenGlobber: c,
		}
	}
	return nil
}

// ChildrenGlobberInvoker returns an Invoker for an object that implements the
// ChildrenGlobber interface, and nothing else.
func ChildrenGlobberInvoker(children ...string) Invoker {
	return ReflectInvokerOrDie(&obj{children})
}

type obj struct {
	children []string
}

func (o obj) GlobChildren__(_ *context.T, call GlobChildrenServerCall, matcher *glob.Element) error {
	sender := call.SendStream()
	for _, v := range o.children {
		if matcher.Match(v) {
			sender.Send(naming.GlobChildrenReplyName{Value: v})
		}
	}
	return nil
}
