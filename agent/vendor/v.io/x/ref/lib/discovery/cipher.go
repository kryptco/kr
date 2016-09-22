// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/nacl/secretbox"

	"v.io/v23/context"
	"v.io/v23/security"

	"v.io/x/ref/lib/security/bcrypter"
)

var (
	errNoCrypter = errors.New("no crypter")

	// errNoPermission is the error returned by decrypt when there is no permission
	// to decrypt an advertisement.
	errNoPermission = errors.New("no permission")
)

// encrypt encrypts the advertisement so that only users who possess blessings
// matching one of the given blessing patterns can decrypt it. Nil patterns
// means no encryption.
func encrypt(ctx *context.T, adinfo *AdInfo, patterns []security.BlessingPattern) error {
	if len(patterns) == 0 {
		adinfo.EncryptionAlgorithm = NoEncryption
		return nil
	}

	var sharedKey [32]byte
	if _, err := rand.Read(sharedKey[:]); err != nil {
		return err
	}

	adinfo.EncryptionAlgorithm = IbeEncryption
	adinfo.EncryptionKeys = make([]EncryptionKey, len(patterns))
	var err error
	for i, pattern := range patterns {
		if adinfo.EncryptionKeys[i], err = wrapSharedKey(ctx, sharedKey, pattern); err != nil {
			return err
		}
	}

	// We only encrypt addresses for now.
	//
	// TODO(jhahn): Revisit the scope of encryption.
	encrypted := make([]string, len(adinfo.Ad.Addresses))
	for i, addr := range adinfo.Ad.Addresses {
		var n [24]byte
		binary.LittleEndian.PutUint64(n[:], uint64(i))
		encrypted[i] = string(secretbox.Seal(nil, []byte(addr), &n, &sharedKey))
	}
	adinfo.Ad.Addresses = encrypted
	return nil
}

// decrypt decrypts the advertisements using a blessings-based crypter
// from the provided context.
//
// TODO(ataly, ashankar, jhahn): Currently we are using the go
// implementation of the 'bn256' pairings library which causes
// the IBE decryption cost to be 42ms. As a result, clients
// processing discovery advertisements ust use conservative timeouts
// of at least 200ms to ensure that the advertisement is decrypted.
// Once we switch to the C implementation of the library, the
// decryption cost, and therefore this timeout, will come down.
func decrypt(ctx *context.T, adinfo *AdInfo) error {
	if adinfo.EncryptionAlgorithm == NoEncryption {
		// Not encrypted.
		return nil
	}

	if adinfo.EncryptionAlgorithm != IbeEncryption {
		return fmt.Errorf("unsupported encryption algorithm: %v", adinfo.EncryptionAlgorithm)
	}

	var sharedKey *[32]byte
	var err error
	for _, key := range adinfo.EncryptionKeys {
		if sharedKey, err = unwrapSharedKey(ctx, key); err == nil {
			break
		}
	}
	if sharedKey == nil {
		return errNoPermission
	}

	// We only encrypt addresses for now.
	//
	// Note that we should not modify the slice element directly here since the
	// underlying plugins may cache advertisements and the next plugin.Scan()
	// may return the already decrypted addresses.
	decrypted := make([]string, len(adinfo.Ad.Addresses))
	for i, encrypted := range adinfo.Ad.Addresses {
		var n [24]byte
		binary.LittleEndian.PutUint64(n[:], uint64(i))
		addr, ok := secretbox.Open(nil, []byte(encrypted), &n, sharedKey)
		if !ok {
			return errors.New("decryption error")
		}
		decrypted[i] = string(addr)
	}
	adinfo.Ad.Addresses = decrypted
	return nil
}

func wrapSharedKey(ctx *context.T, sharedKey [32]byte, pattern security.BlessingPattern) (EncryptionKey, error) {
	crypter := bcrypter.GetCrypter(ctx)
	if crypter == nil {
		return nil, errNoCrypter
	}
	ctext, err := crypter.Encrypt(ctx, pattern, sharedKey[:])
	if err != nil {
		return nil, err
	}
	var wctext bcrypter.WireCiphertext
	ctext.ToWire(&wctext)
	return EncodeWireCiphertext(&wctext), nil
}

func unwrapSharedKey(ctx *context.T, key EncryptionKey) (*[32]byte, error) {
	wctext, err := DecodeWireCiphertext(key)
	if err != nil {
		return nil, err
	}
	var ctext bcrypter.Ciphertext
	ctext.FromWire(*wctext)
	crypter := bcrypter.GetCrypter(ctx)
	if crypter == nil {
		return nil, errNoCrypter
	}
	decrypted, err := crypter.Decrypt(ctx, &ctext)
	if err != nil {
		return nil, err
	}
	if len(decrypted) != 32 {
		return nil, errors.New("shared key decryption error")
	}
	var sharedKey [32]byte
	copy(sharedKey[:], decrypted)
	return &sharedKey, nil
}
