package main

import (
	"testing"
	"unsafe"
)

func TestFindAllObjects(t *testing.T) {
	var sessionHandle CK_SESSION_HANDLE
	ret := C_OpenSession(0, CKF_SERIAL_SESSION, nil, nil, &sessionHandle)
	if ret != CKR_OK {
		t.Fatal("non-OK ret:", ret)
	}

	ret = C_FindObjectsInit(sessionHandle, nil, 0)
	if ret != CKR_OK {
		t.Fatal("non-OK ret:", ret)
	}

	foundCount := new(ULONG)
	objects := make([]CK_OBJECT_HANDLE, 2)
	ret = C_FindObjects(sessionHandle, &objects[0], ULONG(len(objects)), foundCount)
	if ret != CKR_OK {
		t.Fatal("non-OK ret:", ret)
	}
	if *foundCount != 2 {
		t.Fatal("should find 2 objects")
	}

}

func TestFindSpecificObjects(t *testing.T) {
	var sessionHandle CK_SESSION_HANDLE
	ret := C_OpenSession(0, CKF_SERIAL_SESSION, nil, nil, &sessionHandle)
	if ret != CKR_OK {
		t.Fatal("non-OK ret:", ret)
	}

	pubKeyFilter := CKO_PUBLIC_KEY
	privKeyFilter := CKO_PRIVATE_KEY
	templates := []CK_ATTRIBUTE{
		CK_ATTRIBUTE{
			_type:  CKA_CLASS,
			pValue: unsafe.Pointer(&pubKeyFilter),
		},
		CK_ATTRIBUTE{
			_type:  CKA_CLASS,
			pValue: unsafe.Pointer(&privKeyFilter),
		},
	}
	ret = C_FindObjectsInit(sessionHandle, &templates[0], ULONG(len(templates)))
	if ret != CKR_OK {
		t.Fatal("non-OK ret:", ret)
	}

	foundCount := new(ULONG)
	objects := make([]CK_OBJECT_HANDLE, 2)
	ret = C_FindObjects(sessionHandle, &objects[0], ULONG(len(objects)), foundCount)
	if ret != CKR_OK {
		t.Fatal("non-OK ret:", ret)
	}
	if *foundCount != 2 {
		t.Fatal("should find 2 objects")
	}

}
