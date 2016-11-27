// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build openssl

#include <openssl/crypto.h>
#include <openssl/ec.h>
#include <openssl/ecdsa.h>
#include <openssl/x509.h>

void openssl_lock(int mode, int n);

void openssl_locking_callback(int mode, int n, const char* file, int line) {
	openssl_lock(mode, n); }

void openssl_init_locks() {
	CRYPTO_set_locking_callback(openssl_locking_callback);
}

// d2i_ECPrivateKey + ERR_get_error in a single function.
// This is to ensure that the call to ERR_get_error happens in the same thread
// as the call to d2i_ECPrivateKey. If the two were called from Go, the goroutine
// might be pre-empted and rescheduled on another thread leading to an
// inconsistent error.
EC_KEY* openssl_d2i_ECPrivateKey(const unsigned char* data, long len, unsigned long* e) {
	EC_KEY* k = d2i_ECPrivateKey(NULL, &data, len);
	if (k != NULL) {
		*e = 0;
		return k;
	}
	*e = ERR_get_error();
	return NULL;
}

// d2i_EC_PUBKEY + ERR_get_error in a single function.
// This is to ensure that the call to ERR_get_error happens in the same thread
// as the call to d2i_EC_PUBKEY. If the two were called from Go, the goroutine
// might be pre-empted and rescheduled on another thread leading to an
// inconsistent error.
EC_KEY* openssl_d2i_EC_PUBKEY(const unsigned char* data, long len, unsigned long* e) {
	EC_KEY* k = d2i_EC_PUBKEY(NULL, &data, len);
	if (k != NULL) {
		*e = 0;
		return k;
	}
	*e = ERR_get_error();
	return NULL;
}

// ECDSA_do_sign + ERR_get_error in a single function.
// This is to ensure that the call to ERR_get_error happens in the same thread
// as the call to ECDSA_do_sign. If the two were called from Go, the goroutine
// might be pre-empted and rescheduled on another thread leading to an
// inconsistent error.
ECDSA_SIG* openssl_ECDSA_do_sign(const unsigned char* digest, int len, EC_KEY* key, unsigned long* e) {
	ECDSA_SIG* sig = ECDSA_do_sign(digest, len, key);
	if (sig != NULL) {
		*e = 0;
		return sig;
	}
	*e = ERR_get_error();
	return NULL;
}
