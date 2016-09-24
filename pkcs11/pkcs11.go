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
	C_Initialize:  C.CK_C_Initialize(C.C_Initialize),
	C_GetInfo:     C.CK_C_GetInfo(C.C_GetInfo),
	C_GetSlotList: C.CK_C_GetSlotList(C.C_GetSlotList),
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
		manufacturerID: [32]C.uchar{
			C.uchar('K'), C.uchar('r'), C.uchar('y'), C.uchar('p'),
			C.uchar('t'), C.uchar('C'), C.uchar('o'), C.uchar(','),
			C.uchar(' '), C.uchar('I'), C.uchar('n'), C.uchar('c'),
			C.uchar('.'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
		},
		libraryDescription: [32]C.uchar{
			C.uchar('K'), C.uchar('r'), C.uchar('y'), C.uchar('p'),
			C.uchar('t'), C.uchar('C'), C.uchar('o'), C.uchar(','),
			C.uchar(' '), C.uchar('I'), C.uchar('n'), C.uchar('c'),
			C.uchar('.'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
			C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'), C.uchar('\x00'),
		},
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
	*count = 0
	return C.CKR_OK
}

func main() {}
