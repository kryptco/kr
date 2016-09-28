// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

// UniversalServiceMethods defines the set of methods that are implemented on
// all services.
//
// TODO(toddw): Remove this interface now that there aren't any universal
// methods?  Or should we add VDL-generated Signature / MethodSignature / Glob
// methods as a convenience?
type UniversalServiceMethods interface {
}
