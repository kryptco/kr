package main

/*
#include <stdio.h>
#include <stdlib.h>
#include "pkcs11.h"
*/
import (
	"C"
)

//export C_InitToken
func C_InitToken(slot C.CK_SLOT_ID, pin *C.CK_CHAR, pinLen C.CK_ULONG, label *C.CK_CHAR) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_InitToken")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_InitPIN
func C_InitPIN(session C.CK_SESSION_HANDLE, pin *C.CK_CHAR, pinLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_InitPIN")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SetPIN
func C_SetPIN(session C.CK_SESSION_HANDLE, oldPin *C.CK_CHAR, oldPinLen C.CK_ULONG, newPin *C.CK_CHAR, newPinLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SetPIN")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CloseAllSessions
func C_CloseAllSessions(slot C.CK_SLOT_ID) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_CloseAllSessions")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetOperationState
func C_GetOperationState(session C.CK_SESSION_HANDLE, operationState *C.CK_CHAR, operationStateLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_GetOperationState")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SetOperationState
func C_SetOperationState(session C.CK_SESSION_HANDLE, operationState *C.CK_CHAR, operationStateLen C.CK_ULONG, encryptionKey C.CK_OBJECT_HANDLE, authenticationKey C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SetOperationState")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Login
func C_Login(session C.CK_SESSION_HANDLE, userType C.CK_USER_TYPE, pin *C.CK_CHAR, pinLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_Login")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Logout
func C_Logout(session C.CK_SESSION_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_Logout")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CreateObject
func C_CreateObject(session C.CK_SESSION_HANDLE, attributeTemplate *C.CK_ATTRIBUTE, count C.CK_ULONG, object *C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_CreateObject")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CopyObject
func C_CopyObject(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE, template *C.CK_ATTRIBUTE, count C.CK_ULONG, newObject *C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_CopyObject")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DestroyObject
func C_DestroyObject(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DestroyObject")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetObjectSize
func C_GetObjectSize(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE, size *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_GetObjectSize")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SetAttributeValue
func C_SetAttributeValue(session C.CK_SESSION_HANDLE, object C.CK_OBJECT_HANDLE, template *C.CK_ATTRIBUTE, count C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SetAttributeValue")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_EncryptInit
func C_EncryptInit(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_EncryptInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Encrypt
func C_Encrypt(session C.CK_SESSION_HANDLE, data *C.CK_CHAR, dataLen C.CK_ULONG, encryptedData *C.CK_CHAR, encryptedDataLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_Encrypt")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_EncryptUpdate
func C_EncryptUpdate(session C.CK_SESSION_HANDLE, part *C.CK_CHAR, partLen C.CK_ULONG, encryptedPart *C.CK_CHAR, encryptedPartLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_EncryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_EncryptFinal
func C_EncryptFinal(session C.CK_SESSION_HANDLE, lastEncryptedPart *C.CK_CHAR, lastEncryptedPartLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_EncryptFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptInit
func C_DecryptInit(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DecryptInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Decrypt
func C_Decrypt(session C.CK_SESSION_HANDLE, encryptedData *C.CK_CHAR, encryptedDataLen C.CK_ULONG, data *C.CK_CHAR, dataLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_Decrypt")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptUpdate
func C_DecryptUpdate(session C.CK_SESSION_HANDLE, encryptedPart *C.CK_CHAR, encryptedPartLen C.CK_ULONG, part *C.CK_CHAR, partLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DecryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptFinal
func C_DecryptFinal(session C.CK_SESSION_HANDLE, lastPart *C.CK_CHAR, lastPartLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DecryptFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestInit
func C_DigestInit(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DigestInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Digest
func C_Digest(session C.CK_SESSION_HANDLE, data *C.CK_CHAR, dataLen C.CK_ULONG, digest *C.CK_CHAR, digestLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_Digest")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestUpdate
func C_DigestUpdate(session C.CK_SESSION_HANDLE, part *C.CK_CHAR, partLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DigestUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestKey
func C_DigestKey(session C.CK_SESSION_HANDLE, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DigestKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestFinal
func C_DigestFinal(session C.CK_SESSION_HANDLE, digest *C.CK_CHAR, digestLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DigestFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignUpdate
func C_SignUpdate(session C.CK_SESSION_HANDLE, part *C.CK_CHAR, partLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SignUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignFinal
func C_SignFinal(session C.CK_SESSION_HANDLE, signature *C.CK_CHAR, signatureLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SignFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignRecoverInit
func C_SignRecoverInit(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SignRecoverInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignRecover
func C_SignRecover(session C.CK_SESSION_HANDLE, data *C.CK_CHAR, dataLen C.CK_ULONG, signature *C.CK_CHAR, signatureLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SignRecover")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyInit
func C_VerifyInit(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_VerifyInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_Verify
func C_Verify(session C.CK_SESSION_HANDLE, data *C.CK_CHAR, dataLen C.CK_ULONG, signature *C.CK_CHAR, signatureLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_Verify")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyUpdate
func C_VerifyUpdate(session C.CK_SESSION_HANDLE, part *C.CK_CHAR, partLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_VerifyUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyFinal
func C_VerifyFinal(session C.CK_SESSION_HANDLE, signature *C.CK_CHAR, signatureLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_VerifyFinal")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyRecoverInit
func C_VerifyRecoverInit(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, key C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_VerifyRecoverInit")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_VerifyRecover
func C_VerifyRecover(session C.CK_SESSION_HANDLE, signature *C.CK_CHAR, signatureLen C.CK_ULONG, data *C.CK_CHAR, dataLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_VerifyRecover")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DigestEncryptUpdate
func C_DigestEncryptUpdate(session C.CK_SESSION_HANDLE, part *C.CK_CHAR, partLen C.CK_ULONG, encryptedPart *C.CK_CHAR, encryptedPartLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DigestEncryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptDigestUpdate
func C_DecryptDigestUpdate(session C.CK_SESSION_HANDLE, encryptedPart *C.CK_CHAR, encryptedPartLen C.CK_ULONG, part *C.CK_CHAR, partLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DecryptDigestUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SignEncryptUpdate
func C_SignEncryptUpdate(session C.CK_SESSION_HANDLE, part *C.CK_CHAR, partLen C.CK_ULONG, encryptedPart *C.CK_CHAR, encryptedPartLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SignEncryptUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DecryptVerifyUpdate
func C_DecryptVerifyUpdate(session C.CK_SESSION_HANDLE, encryptedPart *C.CK_CHAR, encryptedPartLen C.CK_ULONG, part *C.CK_CHAR, partLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DecryptVerifyUpdate")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GenerateKey
func C_GenerateKey(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, template *C.CK_ATTRIBUTE, count C.CK_ULONG, key *C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_GenerateKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GenerateKeyPair
func C_GenerateKeyPair(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, pubKeyTemplate *C.CK_ATTRIBUTE, pubKeyAttributeCount C.CK_ULONG, privKeyTemplate *C.CK_ATTRIBUTE, privKeyAttributeCount C.CK_ULONG, pubKey *C.CK_OBJECT_HANDLE, privKey *C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_GenerateKeyPair")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_WrapKey
func C_WrapKey(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, wrappingKey C.CK_OBJECT_HANDLE, key C.CK_OBJECT_HANDLE, wrappedKey *C.CK_CHAR, wrappedKeyLen *C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_WrapKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_UnwrapKey
func C_UnwrapKey(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, unwrappingKey C.CK_OBJECT_HANDLE, wrappedKey *C.CK_CHAR, wrappedKeyLen C.CK_ULONG, template *C.CK_ATTRIBUTE, attributeCount C.CK_ULONG, key *C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_UnwrapKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_DeriveKey
func C_DeriveKey(session C.CK_SESSION_HANDLE, mechanism *C.CK_MECHANISM, baseKey C.CK_OBJECT_HANDLE, template *C.CK_ATTRIBUTE, attributeCount C.CK_ULONG, key *C.CK_OBJECT_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_DeriveKey")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_SeedRandom
func C_SeedRandom(session C.CK_SESSION_HANDLE, seed *C.CK_CHAR, seedLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_SeedRandom")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GenerateRandom
func C_GenerateRandom(session C.CK_SESSION_HANDLE, randomData *C.CK_CHAR, randomLen C.CK_ULONG) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_GenerateRandom")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_GetFunctionStatus
func C_GetFunctionStatus(session C.CK_SESSION_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_GetFunctionStatus")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_CancelFunction
func C_CancelFunction(session C.CK_SESSION_HANDLE) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_CancelFunction")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}

//export C_WaitForSlotEvent
func C_WaitForSlotEvent(flags C.CK_FLAGS, slot *C.CK_SLOT_ID, reserved C.CK_VOID_PTR) C.CK_RV {
	log.Error("Unsupported PKCS11 function called: C_WaitForSlotEvent")
	return C.CKR_FUNCTION_NOT_SUPPORTED
}
