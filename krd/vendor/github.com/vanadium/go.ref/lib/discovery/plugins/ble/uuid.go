// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ble

import (
	idiscovery "v.io/x/ref/lib/discovery"
)

const (
	// The Vanadium service base uuid and mask.
	vanadiumUuidBase = "3dd1d5a8-0000-5000-0000-000000000000"
	vanadiumUuidMask = "ffffffff-0000-f000-0000-000000000000"
)

var (
	vanadiumUuidPrefixes = [...]byte{0x3d, 0xd1, 0xd5, 0xa8}
)

func newServiceUuid(interfaceName string) idiscovery.Uuid {
	uuid := idiscovery.NewServiceUUID(interfaceName)
	for i, prefix := range vanadiumUuidPrefixes {
		uuid[i] = prefix
	}
	return uuid
}

func toggleServiceUuid(uuid idiscovery.Uuid) {
	uuid[len(uuid)-1] ^= 0x01
}
