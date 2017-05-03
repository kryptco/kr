#![allow(dead_code, non_snake_case, unused_variables, non_upper_case_globals)]
use pkcs11::*;

use std::io::{stderr, Write, Error};

use syslog;
pub use syslog::{Facility, Severity};


lazy_static! {
    pub static ref logger : Option<Box<syslog::Logger>> = {
        get_logger().or_else(|e| {
            writeln!(&mut stderr(), "error connecting to syslog: {}", e);
            Err(e)
        }).ok()
    };
}

fn get_logger() -> Result<Box<syslog::Logger>, Error> {
    let logger_result = syslog::unix(Facility::LOG_USER);
    logger_result.map_err(|e| {
        writeln!(&mut stderr(), "failed to connect to syslog {}", e);
        e
    })
}

macro_rules! error {
    ( $ ( $ arg : expr ), * ) => { 
        logger.as_ref().map(|l| l.err(&format!($($arg),*)).map_err(|e| {
            writeln!(&mut stderr(), "error logging: {:?}", e);
        }));
    };
}

macro_rules! warning {
    ( $ ( $ arg : expr ), * ) => { 
        logger.as_ref().map(|l| l.warn(&format!($($arg),*)).map_err(|e| {
            writeln!(&mut stderr(), "error logging: {:?}", e);
        }));
    };
}

macro_rules! notice {
    ( $ ( $ arg : expr ), * ) => { 
        use std::io::{stderr, Write};
        logger.as_ref().map(|l| l.notice(&format!($($arg),*)).map_err(|e| {
            writeln!(&mut stderr(), "error logging: {:?}", e);
        }));
    };
}


pub extern "C" fn CK_NOTIFY(session: CK_SESSION_HANDLE,
                            event: CK_NOTIFICATION,
                            application: *mut ::std::os::raw::c_void)
                            -> CK_RV {
    notice!("CK_NOTIFY");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_WaitForSlotEvent(flags: CK_FLAGS,
                                        slot: *mut CK_SLOT_ID,
                                        pReserved: *mut ::std::os::raw::c_void)
                                        -> CK_RV {
    notice!("CK_C_WaitForSlotEvent");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_InitToken(slotID: CK_SLOT_ID,
                                 pin: *mut ::std::os::raw::c_uchar,
                                 pin_len: ::std::os::raw::c_ulong,
                                 label: *mut ::std::os::raw::c_uchar)
                                 -> CK_RV {
    notice!("CK_C_InitToken");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_InitPIN(session: CK_SESSION_HANDLE,
                               pin: *mut ::std::os::raw::c_uchar,
                               pin_len: ::std::os::raw::c_ulong)
                               -> CK_RV {
    notice!("CK_C_InitPIN");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SetPIN(session: CK_SESSION_HANDLE,
                              old_pin: *mut ::std::os::raw::c_uchar,
                              old_len: ::std::os::raw::c_ulong,
                              new_pin: *mut ::std::os::raw::c_uchar,
                              new_len: ::std::os::raw::c_ulong)
                              -> CK_RV {
    notice!("CK_C_SetPIN");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_CloseSession(session: CK_SESSION_HANDLE) -> CK_RV {
    notice!("CK_C_CloseSession");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_CloseAllSessions(slotID: CK_SLOT_ID) -> CK_RV {
    notice!("CK_C_CloseAllSessions");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_GetOperationState(session: CK_SESSION_HANDLE,
                                         operation_state: *mut ::std::os::raw::c_uchar,
                                         operation_state_len: *mut ::std::os::raw::c_ulong)
                                         -> CK_RV {
    notice!("CK_C_GetOperationState");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SetOperationState(session: CK_SESSION_HANDLE,
                                         operation_state: *mut ::std::os::raw::c_uchar,
                                         operation_state_len: ::std::os::raw::c_ulong,
                                         encryption_key: CK_OBJECT_HANDLE,
                                         authentiation_key: CK_OBJECT_HANDLE)
                                         -> CK_RV {
    notice!("CK_C_SetOperationState");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Login(session: CK_SESSION_HANDLE,
                             user_type: CK_USER_TYPE,
                             pin: *mut ::std::os::raw::c_uchar,
                             pin_len: ::std::os::raw::c_ulong)
                             -> CK_RV {
    notice!("CK_C_Login");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Logout(session: CK_SESSION_HANDLE) -> CK_RV {
    notice!("CK_C_Logout");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_CreateObject(session: CK_SESSION_HANDLE,
                                    templ: *mut _CK_ATTRIBUTE,
                                    count: ::std::os::raw::c_ulong,
                                    object: *mut CK_OBJECT_HANDLE)
                                    -> CK_RV {
    notice!("CK_C_CreateObject");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_CopyObject(session: CK_SESSION_HANDLE,
                                  object: CK_OBJECT_HANDLE,
                                  templ: *mut _CK_ATTRIBUTE,
                                  count: ::std::os::raw::c_ulong,
                                  new_object: *mut CK_OBJECT_HANDLE)
                                  -> CK_RV {
    notice!("CK_C_CopyObject");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DestroyObject(session: CK_SESSION_HANDLE,
                                     object: CK_OBJECT_HANDLE)
                                     -> CK_RV {
    notice!("CK_C_DestroyObject");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_GetObjectSize(session: CK_SESSION_HANDLE,
                                     object: CK_OBJECT_HANDLE,
                                     size: *mut ::std::os::raw::c_ulong)
                                     -> CK_RV {
    notice!("CK_C_GetObjectSize");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SetAttributeValue(session: CK_SESSION_HANDLE,
                                         object: CK_OBJECT_HANDLE,
                                         templ: *mut _CK_ATTRIBUTE,
                                         count: ::std::os::raw::c_ulong)
                                         -> CK_RV {
    notice!("CK_C_SetAttributeValue");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_EncryptInit(session: CK_SESSION_HANDLE,
                                   mechanism: *mut _CK_MECHANISM,
                                   key: CK_OBJECT_HANDLE)
                                   -> CK_RV {
    notice!("CK_C_EncryptInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Encrypt(session: CK_SESSION_HANDLE,
                               data: *mut ::std::os::raw::c_uchar,
                               data_len: ::std::os::raw::c_ulong,
                               encrypted_data: *mut ::std::os::raw::c_uchar,
                               encrypted_data_len: *mut ::std::os::raw::c_ulong)
                               -> CK_RV {
    notice!("CK_C_Encrypt");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_EncryptUpdate(session: CK_SESSION_HANDLE,
                                     part: *mut ::std::os::raw::c_uchar,
                                     part_len: ::std::os::raw::c_ulong,
                                     encrypted_part: *mut ::std::os::raw::c_uchar,
                                     encrypted_part_len: *mut ::std::os::raw::c_ulong)
                                     -> CK_RV {
    notice!("CK_C_EncryptUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_EncryptFinal(session: CK_SESSION_HANDLE,
                                    last_encrypted_part: *mut ::std::os::raw::c_uchar,
                                    last_encrypted_part_len: *mut ::std::os::raw::c_ulong)
                                    -> CK_RV {
    notice!("CK_C_EncryptFinal");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DecryptInit(session: CK_SESSION_HANDLE,
                                   mechanism: *mut _CK_MECHANISM,
                                   key: CK_OBJECT_HANDLE)
                                   -> CK_RV {
    notice!("CK_C_DecryptInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Decrypt(session: CK_SESSION_HANDLE,
                               encrypted_data: *mut ::std::os::raw::c_uchar,
                               encrypted_data_len: ::std::os::raw::c_ulong,
                               data: *mut ::std::os::raw::c_uchar,
                               data_len: *mut ::std::os::raw::c_ulong)
                               -> CK_RV {
    notice!("CK_C_Decrypt");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DecryptUpdate(session: CK_SESSION_HANDLE,
                                     encrypted_part: *mut ::std::os::raw::c_uchar,
                                     encrypted_part_len: ::std::os::raw::c_ulong,
                                     part: *mut ::std::os::raw::c_uchar,
                                     part_len: *mut ::std::os::raw::c_ulong)
                                     -> CK_RV {
    notice!("CK_C_DecryptUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DecryptFinal(session: CK_SESSION_HANDLE,
                                    last_part: *mut ::std::os::raw::c_uchar,
                                    last_part_len: *mut ::std::os::raw::c_ulong)
                                    -> CK_RV {
    notice!("CK_C_DecryptFinal");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DigestInit(session: CK_SESSION_HANDLE,
                                  mechanism: *mut _CK_MECHANISM)
                                  -> CK_RV {
    notice!("CK_C_DigestInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Digest(session: CK_SESSION_HANDLE,
                              data: *mut ::std::os::raw::c_uchar,
                              data_len: ::std::os::raw::c_ulong,
                              digest: *mut ::std::os::raw::c_uchar,
                              digest_len: *mut ::std::os::raw::c_ulong)
                              -> CK_RV {
    notice!("CK_C_Digest");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DigestUpdate(session: CK_SESSION_HANDLE,
                                    part: *mut ::std::os::raw::c_uchar,
                                    part_len: ::std::os::raw::c_ulong)
                                    -> CK_RV {
    notice!("CK_C_DigestUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DigestKey(session: CK_SESSION_HANDLE, key: CK_OBJECT_HANDLE) -> CK_RV {
    notice!("CK_C_DigestKey");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DigestFinal(session: CK_SESSION_HANDLE,
                                   digest: *mut ::std::os::raw::c_uchar,
                                   digest_len: *mut ::std::os::raw::c_ulong)
                                   -> CK_RV {
    notice!("CK_C_DigestFinal");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SignInit(session: CK_SESSION_HANDLE,
                                mechanism: *mut _CK_MECHANISM,
                                key: CK_OBJECT_HANDLE)
                                -> CK_RV {
    notice!("CK_C_SignInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Sign(session: CK_SESSION_HANDLE,
                            data: *mut ::std::os::raw::c_uchar,
                            data_len: ::std::os::raw::c_ulong,
                            signature: *mut ::std::os::raw::c_uchar,
                            signature_len: *mut ::std::os::raw::c_ulong)
                            -> CK_RV {
    notice!("CK_C_Sign");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SignUpdate(session: CK_SESSION_HANDLE,
                                  part: *mut ::std::os::raw::c_uchar,
                                  part_len: ::std::os::raw::c_ulong)
                                  -> CK_RV {
    notice!("CK_C_SignUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SignFinal(session: CK_SESSION_HANDLE,
                                 signature: *mut ::std::os::raw::c_uchar,
                                 signature_len: *mut ::std::os::raw::c_ulong)
                                 -> CK_RV {
    notice!("CK_C_SignFinal");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SignRecoverInit(session: CK_SESSION_HANDLE,
                                       mechanism: *mut _CK_MECHANISM,
                                       key: CK_OBJECT_HANDLE)
                                       -> CK_RV {
    notice!("CK_C_SignRecoverInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SignRecover(session: CK_SESSION_HANDLE,
                                   data: *mut ::std::os::raw::c_uchar,
                                   data_len: ::std::os::raw::c_ulong,
                                   signature: *mut ::std::os::raw::c_uchar,
                                   signature_len: *mut ::std::os::raw::c_ulong)
                                   -> CK_RV {
    notice!("CK_C_SignRecover");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_VerifyInit(session: CK_SESSION_HANDLE,
                                  mechanism: *mut _CK_MECHANISM,
                                  key: CK_OBJECT_HANDLE)
                                  -> CK_RV {
    notice!("CK_C_VerifyInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_Verify(session: CK_SESSION_HANDLE,
                              data: *mut ::std::os::raw::c_uchar,
                              data_len: ::std::os::raw::c_ulong,
                              signature: *mut ::std::os::raw::c_uchar,
                              signature_len: ::std::os::raw::c_ulong)
                              -> CK_RV {
    notice!("CK_C_Verify");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_VerifyUpdate(session: CK_SESSION_HANDLE,
                                    part: *mut ::std::os::raw::c_uchar,
                                    part_len: ::std::os::raw::c_ulong)
                                    -> CK_RV {
    notice!("CK_C_VerifyUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_VerifyFinal(session: CK_SESSION_HANDLE,
                                   signature: *mut ::std::os::raw::c_uchar,
                                   signature_len: ::std::os::raw::c_ulong)
                                   -> CK_RV {
    notice!("CK_C_VerifyFinal");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_VerifyRecoverInit(session: CK_SESSION_HANDLE,
                                         mechanism: *mut _CK_MECHANISM,
                                         key: CK_OBJECT_HANDLE)
                                         -> CK_RV {
    notice!("CK_C_VerifyRecoverInit");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_VerifyRecover(session: CK_SESSION_HANDLE,
                                     signature: *mut ::std::os::raw::c_uchar,
                                     signature_len: ::std::os::raw::c_ulong,
                                     data: *mut ::std::os::raw::c_uchar,
                                     data_len: *mut ::std::os::raw::c_ulong)
                                     -> CK_RV {
    notice!("CK_C_VerifyRecover");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DigestEncryptUpdate(session: CK_SESSION_HANDLE,
                                           part: *mut ::std::os::raw::c_uchar,
                                           part_len: ::std::os::raw::c_ulong,
                                           encrypted_part: *mut ::std::os::raw::c_uchar,
                                           encrypted_part_len: *mut ::std::os::raw::c_ulong)
                                           -> CK_RV {
    notice!("CK_C_DigestEncryptUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DecryptDigestUpdate(session: CK_SESSION_HANDLE,
                                           encrypted_part: *mut ::std::os::raw::c_uchar,
                                           encrypted_part_len: ::std::os::raw::c_ulong,
                                           part: *mut ::std::os::raw::c_uchar,
                                           part_len: *mut ::std::os::raw::c_ulong)
                                           -> CK_RV {
    notice!("CK_C_DecryptDigestUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SignEncryptUpdate(session: CK_SESSION_HANDLE,
                                         part: *mut ::std::os::raw::c_uchar,
                                         part_len: ::std::os::raw::c_ulong,
                                         encrypted_part: *mut ::std::os::raw::c_uchar,
                                         encrypted_part_len: *mut ::std::os::raw::c_ulong)
                                         -> CK_RV {
    notice!("CK_C_SignEncryptUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DecryptVerifyUpdate(session: CK_SESSION_HANDLE,
                                           encrypted_part: *mut ::std::os::raw::c_uchar,
                                           encrypted_part_len: ::std::os::raw::c_ulong,
                                           part: *mut ::std::os::raw::c_uchar,
                                           part_len: *mut ::std::os::raw::c_ulong)
                                           -> CK_RV {
    notice!("CK_C_DecryptVerifyUpdate");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_GenerateKey(session: CK_SESSION_HANDLE,
                                   mechanism: *mut _CK_MECHANISM,
                                   templ: *mut _CK_ATTRIBUTE,
                                   count: ::std::os::raw::c_ulong,
                                   key: *mut CK_OBJECT_HANDLE)
                                   -> CK_RV {
    notice!("CK_C_GenerateKey");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_GenerateKeyPair(session: CK_SESSION_HANDLE,
                                       mechanism: *mut _CK_MECHANISM,
                                       public_key_template: *mut _CK_ATTRIBUTE,
                                       public_key_attribute_count: ::std::os::raw::c_ulong,
                                       private_key_template: *mut _CK_ATTRIBUTE,
                                       private_key_attribute_count: ::std::os::raw::c_ulong,
                                       public_key: *mut CK_OBJECT_HANDLE,
                                       private_key: *mut CK_OBJECT_HANDLE)
                                       -> CK_RV {
    notice!("CK_C_GenerateKeyPair");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_WrapKey(session: CK_SESSION_HANDLE,
                               mechanism: *mut _CK_MECHANISM,
                               wrapping_key: CK_OBJECT_HANDLE,
                               key: CK_OBJECT_HANDLE,
                               wrapped_key: *mut ::std::os::raw::c_uchar,
                               wrapped_key_len: *mut ::std::os::raw::c_ulong)
                               -> CK_RV {
    notice!("CK_C_WrapKey");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_UnwrapKey(session: CK_SESSION_HANDLE,
                                 mechanism: *mut _CK_MECHANISM,
                                 unwrapping_key: CK_OBJECT_HANDLE,
                                 wrapped_key: *mut ::std::os::raw::c_uchar,
                                 wrapped_key_len: ::std::os::raw::c_ulong,
                                 templ: *mut _CK_ATTRIBUTE,
                                 attribute_count: ::std::os::raw::c_ulong,
                                 key: *mut CK_OBJECT_HANDLE)
                                 -> CK_RV {
    notice!("CK_C_UnwrapKey");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_DeriveKey(session: CK_SESSION_HANDLE,
                                 mechanism: *mut _CK_MECHANISM,
                                 base_key: CK_OBJECT_HANDLE,
                                 templ: *mut _CK_ATTRIBUTE,
                                 attribute_count: ::std::os::raw::c_ulong,
                                 key: *mut CK_OBJECT_HANDLE)
                                 -> CK_RV {
    notice!("CK_C_DeriveKey");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_SeedRandom(session: CK_SESSION_HANDLE,
                                  seed: *mut ::std::os::raw::c_uchar,
                                  seed_len: ::std::os::raw::c_ulong)
                                  -> CK_RV {
    notice!("CK_C_SeedRandom");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_GenerateRandom(session: CK_SESSION_HANDLE,
                                      random_data: *mut ::std::os::raw::c_uchar,
                                      random_len: ::std::os::raw::c_ulong)
                                      -> CK_RV {
    notice!("CK_C_GenerateRandom");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_GetFunctionStatus(session: CK_SESSION_HANDLE) -> CK_RV {
    notice!("CK_C_GetFunctionStatus");
    CKR_FUNCTION_NOT_SUPPORTED
}
pub extern "C" fn CK_C_CancelFunction(session: CK_SESSION_HANDLE) -> CK_RV {
    notice!("CK_C_CancelFunction");
    CKR_FUNCTION_NOT_SUPPORTED
}
