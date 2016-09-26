package main

/*
#include <stdio.h>
#include <stdlib.h>
#include "pkcs11.h"
*/
import (
	"C"
)
import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	//"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"sync"
	"unsafe"
)

func log(s string) {
	f, _ := os.OpenFile("/tmp/pkcs11spy.log", os.O_RDWR|os.O_APPEND, 0660)
	f.WriteString(s + "\n")
	f.Close()
}

var functions C.CK_FUNCTION_LIST = C.CK_FUNCTION_LIST{
	version: C.struct__CK_VERSION{
		major: 0,
		minor: 1,
	},
	C_Initialize:        C.CK_C_Initialize(C.C_Initialize),
	C_GetInfo:           C.CK_C_GetInfo(C.C_GetInfo),
	C_GetSlotList:       C.CK_C_GetSlotList(C.C_GetSlotList),
	C_GetTokenInfo:      C.CK_C_GetTokenInfo(C.C_GetTokenInfo),
	C_OpenSession:       C.CK_C_OpenSession(C.C_OpenSession),
	C_CloseSession:      C.CK_C_CloseSession(C.C_CloseSession),
	C_FindObjectsInit:   C.CK_C_FindObjectsInit(C.C_FindObjectsInit),
	C_FindObjects:       C.CK_C_FindObjects(C.C_FindObjects),
	C_FindObjectsFinal:  C.CK_C_FindObjectsFinal(C.C_FindObjectsFinal),
	C_GetAttributeValue: C.CK_C_GetAttributeValue(C.C_GetAttributeValue),
	C_SignInit:          C.CK_C_SignInit(C.C_SignInit),
	C_Sign:              C.CK_C_Sign(C.C_Sign),
	C_Finalize:          C.CK_C_Finalize(C.C_Finalize),
}

//export C_GetFunctionList
func C_GetFunctionList(l **C.CK_FUNCTION_LIST) C.CK_RV {
	*l = &functions
	//return C.CKR_GENERAL_ERROR
	return C.CKR_OK
}

//export C_Initialize
func C_Initialize(C.CK_VOID_PTR) C.CK_RV {
	return C.CKR_OK
}

//export C_GetInfo
func C_GetInfo(ck_info *C.CK_INFO) C.CK_RV {
	*ck_info = C.CK_INFO{
		cryptokiVersion: C.struct__CK_VERSION{
			major: 2,
			minor: 20,
		},
		manufacturerID:     bytesToChar32([]byte("KryptCo Inc.")),
		libraryDescription: bytesToChar32([]byte("kryptonite pkcs11 middleware")),
		libraryVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		flags: 0,
	}
	return C.CKR_OK
}

//export C_GetSlotList
func C_GetSlotList(token_present C.uchar, slot_list *C.CK_SLOT_ID, count *C.ulong) C.CK_RV {
	if *count == 0 {
		*count = 1
		return C.CKR_OK
	}
	*count = 1
	*slot_list = 0
	return C.CKR_OK
}

//export C_GetTokenInfo
func C_GetTokenInfo(slotID C.CK_SLOT_ID, tokenInfo *C.CK_TOKEN_INFO) C.CK_RV {
	*tokenInfo = C.CK_TOKEN_INFO{
		label:               bytesToChar32([]byte("iOS")),
		manufacturerID:      bytesToChar32([]byte("KryptCo Inc.")),
		ulMaxSessionCount:   16,
		ulSessionCount:      0,
		ulMaxRwSessionCount: 16,
		ulRwSessionCount:    0,
		ulMaxPinLen:         0,
		ulMinPinLen:         0,
	}
	return C.CKR_OK
}

//export C_OpenSession
func C_OpenSession(slotID C.CK_SLOT_ID, flags C.CK_FLAGS, pApplication C.CK_VOID_PTR,
	Notify C.CK_NOTIFY, sessionHandle C.CK_SESSION_HANDLE_PTR) C.CK_RV {
	return C.CKR_OK
}

//export C_CloseSession
func C_CloseSession(session C.CK_SESSION_HANDLE) C.CK_RV {
	mutex.Lock()
	defer mutex.Unlock()
	delete(sessionSentPk, session)
	return C.CKR_OK
}

var sessionFindObjectTypes map[C.CK_SESSION_HANDLE][]C.CK_ATTRIBUTE = map[C.CK_SESSION_HANDLE][]C.CK_ATTRIBUTE{}
var mutex sync.Mutex

//export C_FindObjectsInit
func C_FindObjectsInit(session C.CK_SESSION_HANDLE, templates C.CK_ATTRIBUTE_PTR, count C.CK_ULONG) C.CK_RV {
	mutex.Lock()
	defer mutex.Unlock()
	attributes := []C.CK_ATTRIBUTE{}
	log(fmt.Sprintf("count %d", count))
	for i := C.CK_ULONG(0); i < count; i++ {
		attributes = append(attributes, *templates)
		templates = C.CK_ATTRIBUTE_PTR(unsafe.Pointer(uintptr(unsafe.Pointer(templates)) + unsafe.Sizeof(*templates)))
	}
	if len(attributes) > 0 {
		sessionFindObjectTypes[session] = attributes
	}
	return C.CKR_OK
}

const PUBKEY_HANDLE C.CK_OBJECT_HANDLE = 1
const PRIVKEY_HANDLE C.CK_OBJECT_HANDLE = 2

var PUBKEY_ID []byte = []byte{1}

var sessionSentPk map[C.CK_SESSION_HANDLE]bool = map[C.CK_SESSION_HANDLE]bool{}

//export C_FindObjects
func C_FindObjects(session C.CK_SESSION_HANDLE, objects C.CK_OBJECT_HANDLE_PTR, maxCount C.CK_ULONG, count C.CK_ULONG_PTR) C.CK_RV {
	mutex.Lock()
	defer mutex.Unlock()
	attributes, ok := sessionFindObjectTypes[session]
	if !ok || maxCount == 0 {
		return C.CKR_GENERAL_ERROR
	}
	log(fmt.Sprintf("count %d maxCount %d", *count, maxCount))
	foundModulus := false
	foundPublicExponent := false
	for _, attribute := range attributes {
		switch attribute._type {
		case C.CKA_CLASS:
			class := C.CK_OBJECT_CLASS_PTR(attribute.pValue)
			if *class == C.CKO_PUBLIC_KEY {
				if sent, ok := sessionSentPk[session]; ok && sent {
					*count = C.CK_ULONG(0)
					return C.CKR_OK
				}
				*count = C.CK_ULONG(1)
				*objects = PUBKEY_HANDLE
				return C.CKR_OK
			}
			if *class == C.CKO_PRIVATE_KEY {
				*count = C.CK_ULONG(1)
				*objects = PRIVKEY_HANDLE
				return C.CKR_OK
			}
		case C.CKA_KEY_TYPE:
			log(fmt.Sprintf("found key type %d", *C.CK_ULONG_PTR(attribute.pValue)))
		}
	}
	if foundModulus && foundPublicExponent {
		log(fmt.Sprintf("found rsa"))
		*count = 1
	} else {
		*count = 0
	}
	return C.CKR_OK
}

//export C_FindObjectsFinal
func C_FindObjectsFinal(session C.CK_SESSION_HANDLE) C.CK_RV {
	return C.CKR_OK
}

var sk = parse()
var pk = &sk.PublicKey

func gen() *rsa.PrivateKey {
	sk, _ := rsa.GenerateKey(rand.Reader, 4096)
	return sk
}

func parse() *rsa.PrivateKey {
	_ = "MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA0fAZp+DuQKltrL5b0NPY9awpDVbg4aEedPKsAGReE1d/m96OvlswV5WOjd9sz7Qr0q1WxM+LHbIiORRLrEunHaSdkICVWc7SLV8LI/vsxIs+x8w/2llreutAVFBwhU5I4SK9bFdlDu1BTxQi83oRiM2oECqOZd34qCww16TmnSCLKUeRDigB4bSwgav807BB+wDi5Pg6FneI41XyQY+TaMtEm+h3fxnE+J+2XlG4tuwAv7n2N4lN2gsl2b1PITtQgzeeHRjpDKFVfhUApacCIu3Ia8kaQXDKC6zCBCk8pbWcLtrp35a8G/WPqgxvvVsGrWHmY1gtTwVhOYk5AtkaUjGudWspoTRO5lB59IGNhsr4xcSwK/SbxgYelB/Lj7GLIuxUZLwRZm+jjK7BlKg5883YrwZmTg5BFcjOLw7phbygrPyf7HzUMFyZaBr5dLN5m5nzUs1lxIY/moRkmcZKsxPOfh2DO91kdess7U6/wXowfB3OS1jme2cpefX8pTfxfVLZJxf7Qpll6PZLpMyg5zLnEIkvzwicHK0CJeA94p6eaXtO53li3psrYRvRrxAS5TkyHOR6//EOfxsBLol7jHpAkMEN6ljs9uivSEH/TYW+itde10StIZ36IXmJsHvDEi6AqM01QGz4aI55V9zLk7GkiJOVh3IueAuAvlt7syMCAwEAAQ=="
	rsaSkB64 := "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"
	der, _ := base64.StdEncoding.DecodeString(rsaSkB64)
	sk, _ := x509.ParsePKCS1PrivateKey(der)
	return sk
}

//export C_GetAttributeValue
func C_GetAttributeValue(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE, template C.CK_ATTRIBUTE_PTR, count C.CK_ULONG) C.CK_RV {
	skDER := x509.MarshalPKCS1PrivateKey(sk)
	pkDER, _ := x509.MarshalPKIXPublicKey(&sk.PublicKey)
	log("private " + base64.StdEncoding.EncodeToString(skDER))
	log("public " + base64.StdEncoding.EncodeToString(pkDER))

	sshPk, err := ssh.NewPublicKey(pk)
	if err != nil {
		log("ssh pk err: " + err.Error())
	} else {
		log(sshPk.Type() + " " + base64.StdEncoding.EncodeToString(sshPk.Marshal()))
	}

	templateIter := template
	modulus := pk.N.Bytes()
	eBytes := &bytes.Buffer{}
	err = binary.Write(eBytes, binary.BigEndian, int64(pk.E))
	if err != nil {
		log("public exponent binary encoding error: " + err.Error())
		return C.CKR_GENERAL_ERROR
	}
	e := eBytes.Bytes()
	for i := C.CK_ULONG(0); i < count; i++ {
		switch (*templateIter)._type {
		case C.CKA_ID:
			(*templateIter).pValue = unsafe.Pointer(C.CBytes(PUBKEY_ID))
			(*templateIter).ulValueLen = C.ulong(len(PUBKEY_ID))
		case C.CKA_MODULUS:
			(*templateIter).pValue = unsafe.Pointer(C.CBytes(modulus))
			(*templateIter).ulValueLen = C.ulong(len(modulus))
		case C.CKA_PUBLIC_EXPONENT:
			(*templateIter).pValue = unsafe.Pointer(C.CBytes(e))
			(*templateIter).ulValueLen = C.ulong(len(e))
		}
		templateIter = C.CK_ATTRIBUTE_PTR(unsafe.Pointer(uintptr(unsafe.Pointer(templateIter)) + unsafe.Sizeof(*template)))
	}
	sessionSentPk[session] = true
	return C.CKR_OK
}

//export C_SignInit
func C_SignInit(session C.CK_SESSION_HANDLE, mechanism C.CK_MECHANISM_PTR, key C.CK_OBJECT_HANDLE) C.CK_RV {
	return C.CKR_OK
}

//export C_Sign
func C_Sign(session C.CK_SESSION_HANDLE,
	data C.CK_BYTE_PTR, dataLen C.ulong,
	signature C.CK_BYTE_PTR, signatureLen *C.ulong) C.CK_RV {
	sshSigner, err := ssh.NewSignerFromSigner(sk)
	_ = sshSigner
	if err != nil {
		log("ssh signer error: " + err.Error())
		return C.CKR_GENERAL_ERROR
	}
	message := C.GoBytes(unsafe.Pointer(data), C.int(dataLen))
	//sig, err := sshSigner.Sign(rand.Reader, message)
	//sigBytes := sig.Blob
	//digest := sha256.Sum256(message)
	//sigBytes, err := sk.Sign(rand.Reader, digest[:], crypto.SHA256)
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, sk, crypto.Hash(0), message)
	if err != nil {
		log("sig error: " + err.Error())
		return C.CKR_GENERAL_ERROR
	} else {
		for _, b := range sigBytes {
			*signature = C.CK_BYTE(b)
			signature = C.CK_BYTE_PTR(unsafe.Pointer(uintptr(unsafe.Pointer(signature)) + 1))
		}
		*signatureLen = C.ulong(len(sigBytes))
		log("set sig")
	}
	return C.CKR_OK
}

//export C_Finalize
func C_Finalize(reserved C.CK_VOID_PTR) C.CK_RV {
	return C.CKR_OK
}

func bytesToChar32(b []byte) [32]C.uchar {
	for len(b) < 32 {
		b = append(b, byte(0))
	}
	return [32]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
		C.uchar(b[16]), C.uchar(b[17]), C.uchar(b[18]), C.uchar(b[19]),
		C.uchar(b[20]), C.uchar(b[21]), C.uchar(b[22]), C.uchar(b[23]),
		C.uchar(b[24]), C.uchar(b[25]), C.uchar(b[26]), C.uchar(b[27]),
		C.uchar(b[28]), C.uchar(b[29]), C.uchar(b[30]), C.uchar(b[31]),
	}
}

func bytesToChar16(b []byte) [16]C.uchar {
	for len(b) < 16 {
		b = append(b, byte(0))
	}
	return [16]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
	}
}

func main() {}
