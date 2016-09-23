// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ble

import (
	"bytes"
	"errors"
	"fmt"

	"v.io/v23/discovery"

	idiscovery "v.io/x/ref/lib/discovery"
)

const (
	// We pack an advertisement except attachments into a few characteristics
	// with values up to 512 byte. This makes it simple to handle the limit on
	// the maximum size of each characteristic value. And it would be more
	// efficient by sending the smaller number of packets.

	// The maximum size of a characteristic value is limited to 512 bytes
	// by the specification.
	//
	// See Bluetooth specification 4.2, section 3.2.9:
	// https://www.bluetooth.com/specifications/adopted-specifications
	maxCharacteristicValueLen = 512

	// Format string for packed characteristic uuids.
	packedCharacteristicUuidFmt = "31ca10d5-0195-54fa-9344-25fcd7072e%x%x"

	// We should allow up to 16 packed characteristics to keep the packed
	// characteristic uuids valid. This should be enough since we limit
	// the size of advertisement except attachments to 512 bytes.
	maxNumPackedServices                  = 16
	maxNumPackedCharacteristicsPerService = 16
)

func cUuid(i, j int) string {
	return fmt.Sprintf(packedCharacteristicUuidFmt, i, j)
}

func packToCharacteristics(encoded map[discovery.AdId][]byte) map[string][]byte {
	cs := make(map[string][]byte)

	i := 0
	for _, v := range encoded {
		buf := bytes.NewBuffer(v)
		for j := 0; buf.Len() > 0; j++ {
			cs[cUuid(i, j)] = buf.Next(maxCharacteristicValueLen)
		}
		i++
	}
	return cs
}

func encodeAdInfo(adinfo *idiscovery.AdInfo) ([]byte, error) {
	// The current encoding format is
	//
	//	<Id>
	//	<InterfaceName>
	//	<Addresses encoded using idiscovery.PackAddresses>
	//	<#Attributes>[<AttributeKey><AttributeValue>...]
	//	<EncryptionAlgorithm>[<#EncryptionKeys><EncryptionKey>...]
	//	<Hash>
	//	<TimestampNs>
	//      <DirAddrs encoded using idiscovery.PackAddresses>
	//	<Status>
	//
	// Any change of this format (except appending new fields) would break decoding.
	// We can handle any versioning through different characteristic uuids if needed.
	buf := idiscovery.NewEncodingBuffer(nil)

	buf.Write(adinfo.Ad.Id[:])
	buf.WriteString(adinfo.Ad.InterfaceName)

	buf.WriteBytes(idiscovery.PackAddresses(adinfo.Ad.Addresses))

	buf.WriteInt(len(adinfo.Ad.Attributes))
	for k, v := range adinfo.Ad.Attributes {
		buf.WriteString(k)
		buf.WriteString(v)
	}

	buf.WriteInt(int(adinfo.EncryptionAlgorithm))
	if adinfo.EncryptionAlgorithm != idiscovery.NoEncryption {
		buf.WriteInt(len(adinfo.EncryptionKeys))
		for _, key := range adinfo.EncryptionKeys {
			buf.WriteBytes(key)
		}
	}

	buf.Write(adinfo.Hash[:])
	buf.Write(idiscovery.EncodeTimestamp(adinfo.TimestampNs))

	if len(adinfo.Ad.Attachments) == 0 {
		buf.WriteBytes(nil) // No DirAddrs necessary
		buf.WriteInt(int(idiscovery.AdReady))
	} else {
		buf.WriteBytes(idiscovery.PackAddresses(adinfo.DirAddrs))
		buf.WriteInt(int(idiscovery.AdPartiallyReady))
	}

	if buf.Len() > maxCharacteristicValueLen*maxNumPackedCharacteristicsPerService {
		return nil, errors.New("max advertisement size exceeded")
	}
	return buf.Bytes(), nil
}

func unpackFromCharacteristics(cs map[string][]byte) ([][]byte, error) {
	var unpacked [][]byte
	used := 0
	for i := 0; i < maxNumPackedServices; i++ {
		if _, ok := cs[cUuid(i, 0)]; !ok {
			break
		}

		var splitted [][]byte
		for j := 0; j < maxNumPackedCharacteristicsPerService; j++ {
			uuid := cUuid(i, j)
			c, ok := cs[uuid]
			if !ok {
				break
			}
			splitted = append(splitted, c)
			used++
		}

		if len(splitted) == 1 {
			// Short-cut for a single characteristic.
			unpacked = append(unpacked, splitted[0])
		} else {
			n := 0
			for j, d := range splitted {
				if j < len(splitted)-1 && len(d) != maxCharacteristicValueLen {
					return nil, fmt.Errorf("invalid characteristics: %v", cs)
				}
				n += len(d)
			}
			merged := make([]byte, n)
			i := 0
			for _, v := range splitted {
				i += copy(merged[i:], v)
			}
			unpacked = append(unpacked, merged)
		}
	}
	if len(cs) != used {
		return nil, fmt.Errorf("invalid characteristics: %v", cs)
	}
	return unpacked, nil
}

func decodeAdInfo(encoded []byte) (*idiscovery.AdInfo, error) {
	var (
		err error
		buf *idiscovery.EncodingBuffer = idiscovery.NewEncodingBuffer(encoded)

		read = func(p []byte) {
			if err != nil {
				return
			}
			err = buf.Read(p)
		}
		readInt = func() (x int) {
			if err != nil {
				return
			}
			x, err = buf.ReadInt()
			return
		}
		readBytes = func() (p []byte) {
			if err != nil {
				return
			}
			p, err = buf.ReadBytes()
			return
		}
		readString = func() (s string) {
			if err != nil {
				return
			}
			s, err = buf.ReadString()
			return
		}
		readTimestamp = func() (ts int64) {
			if err != nil {
				return
			}
			ts, err = idiscovery.DecodeTimestamp(buf.Next(8))
			return
		}
	)

	adinfo := &idiscovery.AdInfo{}

	read(adinfo.Ad.Id[:])
	adinfo.Ad.InterfaceName = readString()
	adinfo.Ad.Addresses, err = idiscovery.UnpackAddresses(readBytes())

	if n := readInt(); n > 0 {
		adinfo.Ad.Attributes = make(map[string]string, n)
		for i := 0; i < n; i++ {
			adinfo.Ad.Attributes[readString()] = readString()
		}
	}

	adinfo.EncryptionAlgorithm = idiscovery.EncryptionAlgorithm(readInt())
	if adinfo.EncryptionAlgorithm != idiscovery.NoEncryption {
		n := readInt()
		adinfo.EncryptionKeys = make([]idiscovery.EncryptionKey, n)
		for i := 0; i < n; i++ {
			adinfo.EncryptionKeys[i] = readBytes()
		}
	}

	read(adinfo.Hash[:])
	adinfo.TimestampNs = readTimestamp()

	adinfo.DirAddrs, err = idiscovery.UnpackAddresses(readBytes())
	adinfo.Status = idiscovery.AdStatus(readInt())

	if err != nil {
		return nil, err
	}
	return adinfo, nil
}
