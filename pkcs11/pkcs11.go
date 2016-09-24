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
	//"fmt"
	"os"
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
	C_Initialize:   C.CK_C_Initialize(C.C_Initialize),
	C_GetInfo:      C.CK_C_GetInfo(C.C_GetInfo),
	C_GetSlotList:  C.CK_C_GetSlotList(C.C_GetSlotList),
	C_GetTokenInfo: C.CK_C_GetTokenInfo(C.C_GetTokenInfo),
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
