// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"github.com/pborman/uuid"
)

var (
	// UUID of Vanadium namespace.
	// Generated from UUID5("00000000-0000-0000-0000-000000000000", "v.io").
	v23Uuid uuid.UUID = uuid.UUID{0x3d, 0xd1, 0xd5, 0xa8, 0x2e, 0xef, 0x58, 0x16, 0xa7, 0x20, 0xf8, 0x8b, 0x9b, 0xcf, 0x6e, 0xe4}

	// Generated from UUID5("00000000-0000-0000-0000-000000000000", "v.io/attrs").
	v23AttrUuid uuid.UUID = uuid.UUID{0x94, 0x2b, 0x61, 0x64, 0x12, 0x79, 0x5e, 0xb6, 0xb6, 0x43, 0xc9, 0x0c, 0x4c, 0xcc, 0x8a, 0x72}
)

// NewServiceUUID returns a version 5 UUID for the given interface name.
//
// TODO(jhahn): rename to NewServiceUuid.
func NewServiceUUID(interfaceName string) Uuid {
	return Uuid(uuid.NewSHA1(v23Uuid, []byte(interfaceName)))
}

// NewAttributeUUID returns a version 5 UUID for the given name.
//
// TODO(jhahn): rename to NewAttributeUuid.
func NewAttributeUUID(name string) Uuid {
	return Uuid(uuid.NewSHA1(v23AttrUuid, []byte(name)))
}

// String returns the string form of uuid 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'.
func (u Uuid) String() string {
	return uuid.UUID(u).String()
}
