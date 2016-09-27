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
	"log"
)

//export C_GetMechanismList
func C_GetMechanismList() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GetMechanismList")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetMechanismInfo
func C_GetMechanismInfo() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GetMechanismInfo")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_InitToken
func C_InitToken() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_InitToken")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_InitPIN
func C_InitPIN() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_InitPIN")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SetPIN
func C_SetPIN() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SetPIN")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CloseAllSessions
func C_CloseAllSessions() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_CloseAllSessions")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetSessionInfo
func C_GetSessionInfo() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GetSessionInfo")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetOperationState
func C_GetOperationState() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GetOperationState")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SetOperationState
func C_SetOperationState() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SetOperationState")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Login
func C_Login() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_Login")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Logout
func C_Logout() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_Logout")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CreateObject
func C_CreateObject() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_CreateObject")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CopyObject
func C_CopyObject() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_CopyObject")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DestroyObject
func C_DestroyObject() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DestroyObject")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetObjectSize
func C_GetObjectSize() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GetObjectSize")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SetAttributeValue
func C_SetAttributeValue() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SetAttributeValue")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_EncryptInit
func C_EncryptInit() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_EncryptInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Encrypt
func C_Encrypt() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_Encrypt")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_EncryptUpdate
func C_EncryptUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_EncryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_EncryptFinal
func C_EncryptFinal() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_EncryptFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptInit
func C_DecryptInit() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DecryptInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Decrypt
func C_Decrypt() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_Decrypt")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptUpdate
func C_DecryptUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DecryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptFinal
func C_DecryptFinal() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DecryptFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestInit
func C_DigestInit() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DigestInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Digest
func C_Digest() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_Digest")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestUpdate
func C_DigestUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DigestUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestKey
func C_DigestKey() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DigestKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestFinal
func C_DigestFinal() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DigestFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignUpdate
func C_SignUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SignUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignFinal
func C_SignFinal() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SignFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignRecoverInit
func C_SignRecoverInit() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SignRecoverInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignRecover
func C_SignRecover() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SignRecover")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyInit
func C_VerifyInit() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_VerifyInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Verify
func C_Verify() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_Verify")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyUpdate
func C_VerifyUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_VerifyUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyFinal
func C_VerifyFinal() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_VerifyFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyRecoverInit
func C_VerifyRecoverInit() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_VerifyRecoverInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyRecover
func C_VerifyRecover() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_VerifyRecover")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestEncryptUpdate
func C_DigestEncryptUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DigestEncryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptDigestUpdate
func C_DecryptDigestUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DecryptDigestUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignEncryptUpdate
func C_SignEncryptUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SignEncryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptVerifyUpdate
func C_DecryptVerifyUpdate() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DecryptVerifyUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GenerateKey
func C_GenerateKey() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GenerateKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GenerateKeyPair
func C_GenerateKeyPair() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GenerateKeyPair")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_WrapKey
func C_WrapKey() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_WrapKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_UnwrapKey
func C_UnwrapKey() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_UnwrapKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DeriveKey
func C_DeriveKey() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_DeriveKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SeedRandom
func C_SeedRandom() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_SeedRandom")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GenerateRandom
func C_GenerateRandom() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GenerateRandom")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetFunctionStatus
func C_GetFunctionStatus() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_GetFunctionStatus")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CancelFunction
func C_CancelFunction() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_CancelFunction")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_WaitForSlotEvent
func C_WaitForSlotEvent() C.CK_RV {
	log.Println("Unsupported PKCS11 function called: C_WaitForSlotEvent")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}
