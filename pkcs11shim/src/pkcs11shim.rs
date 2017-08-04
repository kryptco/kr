#![allow(non_snake_case, unused_variables, non_upper_case_globals)]

use std::sync::atomic::AtomicUsize;
use std::sync::atomic::Ordering::SeqCst;
use std::sync::Mutex;
use std::env;
use std::fs::OpenOptions;
use std::os::unix::io::AsRawFd;
use std::path::Path;

extern crate libc;
extern crate users;

use self::users::get_user_by_name;
use self::users::os::unix::UserExt;

use pkcs11_unused::*;
use pkcs11::*;
use utils::*;

lazy_static! {
    static ref OLD_STDERR_FD: Mutex<Option<libc::c_int>> = Mutex::new(None);
}

#[no_mangle]
pub extern "C" fn C_GetFunctionList(function_list: *mut *mut _CK_FUNCTION_LIST) -> CK_RV {
    notice!("C_GetFunctionList");
    unsafe {
        *function_list = &mut FUNCTION_LIST;
    }
    CKR_OK
}

pub extern "C" fn CK_C_GetFunctionList(function_list: *mut *mut _CK_FUNCTION_LIST) -> CK_RV {
    notice!("CK_C_GetFunctionList");
    C_GetFunctionList(function_list)
}

/// Symlink original $SSH_AUTH_SOCK to ~/.kr/original-agent.sock
/// Set $SSH_AUTH_SOCK to krd ssh-agent socket
/// Temporarily redirect STDERR to /dev/null to prevent "no keys" error message on older OpenSSH
/// clients
#[allow(unused_must_use)]
extern "C" fn CK_C_Initialize(init_args: *mut ::std::os::raw::c_void) -> CK_RV {
    notice!("CK_C_Initialize");

    let mut krd_auth_sock = if let Ok(sudo_user) = env::var("SUDO_USER") {
        get_user_by_name(&sudo_user).map(|u| u.home_dir().to_path_buf())
    } else {
        env::home_dir()
    };
    if let Some(mut krd_auth_sock) = krd_auth_sock {
        krd_auth_sock.push(".kr/krd-agent.sock");
        if let Ok(original_auth_sock) = env::var("SSH_AUTH_SOCK") {
            if let Some(mut backup_agent) = env::home_dir() {
                use std::os::unix::fs::symlink;
                use std::fs;

                backup_agent.push(".kr/original-agent.sock");
                fs::remove_file(backup_agent.clone());
                if Path::new(&original_auth_sock) != krd_auth_sock {
                    notice!("found backup auth_sock {}", original_auth_sock);
                    match symlink(original_auth_sock, backup_agent) {
                        Err(e) => {
                            error!("error linking backup agent: {:?}", e);
                        },
                        _ => {},
                    };
                }
            }
        }

        env::set_var("SSH_AUTH_SOCK", krd_auth_sock);
    }

    if let Ok(dev_null) = OpenOptions::new()
        .read(true)
        .write(true)
        .open("/dev/null") {
            match OLD_STDERR_FD.lock() {
                Ok(mut old_stderr_fd) => {
                    unsafe {
                        *old_stderr_fd = Some(libc::dup(libc::STDERR_FILENO));
                        libc::dup2(dev_null.as_raw_fd(), libc::STDERR_FILENO);
                    };
                },
                Err(_) => {},
            };
        }
    CKR_OK
}

pub extern "C" fn CK_C_GetInfo(info: *mut _CK_INFO) -> CK_RV {
    notice!("CK_C_GetInfo");
    unsafe {
        *info = CK_INFO {
            cryptokiVersion: CK_VERSION {
                major: 2,
                minor: 20,
            },
            manufacturerID: str_to_char32("KryptCo Inc."),
            flags: 0,
            libraryDescription: str_to_char32("Kryptonite PKCS11 Middleware"),
            libraryVersion: CK_VERSION {
                major: 1,
                minor: 0,
            },
        };
    }
    CKR_OK
}

pub extern "C" fn CK_C_GetSlotList(token_present: ::std::os::raw::c_uchar,
                                   slot_list: *mut CK_SLOT_ID,
                                   count: *mut ::std::os::raw::c_ulong)
                                   -> CK_RV {
    notice!("CK_C_GetSlotList");
    if slot_list.is_null() {
        notice!("slot_list null");
        unsafe {
            *count = 1;
        }
        CKR_OK
    } else {
        if unsafe { *count < 1 } {
            error!("buffer too small");
            CKR_BUFFER_TOO_SMALL
        } else {
            unsafe {
                *count = 1;
                *slot_list = 1;
            }
            CKR_OK
        }
    }
}

pub extern "C" fn CK_C_GetSlotInfo(slotID: CK_SLOT_ID, info: *mut _CK_SLOT_INFO) -> CK_RV {
    notice!("CK_C_GetSlotInfo");
    unsafe {
        *info = CK_SLOT_INFO {
            slotDescription: str_to_char64("Kryptonite PKCS11 Middleware"),
            manufacturerID: str_to_char32("KryptCo, Inc."),
            flags: CKF_TOKEN_PRESENT | CKF_REMOVABLE_DEVICE,
            hardwareVersion: CK_VERSION {
                major: 1,
                minor: 0,
            },
            firmwareVersion: CK_VERSION {
                major: 1,
                minor: 0,
            },
        };
    }
    CKR_OK
}

pub extern "C" fn CK_C_GetTokenInfo(slotID: CK_SLOT_ID, info: *mut _CK_TOKEN_INFO) -> CK_RV {
    notice!("CK_C_GetTokenInfo");
    unsafe {
        *info = CK_TOKEN_INFO {
            label: str_to_char32("Kryptonite iOS"),
            manufacturerID: str_to_char32("KryptCo, Inc."),
            model: str_to_char16("Kryptonite iOS"),
            serialNumber: str_to_char16("1"),
            flags: CKF_TOKEN_INITIALIZED,
            ulMaxSessionCount: 16,
            ulSessionCount: 0,
            ulMaxRwSessionCount: 16,
            ulRwSessionCount: 0,
            ulMaxPinLen: 0,
            ulMinPinLen: 0,
            ulTotalPublicMemory: 0,
            ulFreePublicMemory: 0,
            ulTotalPrivateMemory: 0,
            ulFreePrivateMemory: 0,
            hardwareVersion: CK_VERSION {
                major: 1,
                minor: 0,
            },
            firmwareVersion: CK_VERSION {
                major: 1,
                minor: 0,
            },
            utcTime: str_to_char16(""),
        };
    }
    CKR_OK
}

static MECHANISM_LIST: &'static [CK_MECHANISM_TYPE] = &[CKM_RSA_PKCS, CKM_SHA256_RSA_PKCS];

pub extern "C" fn CK_C_GetMechanismList(slotID: CK_SLOT_ID,
                                        mechanism_list: *mut CK_MECHANISM_TYPE,
                                        count: *mut ::std::os::raw::c_ulong)
                                        -> CK_RV {
    notice!("CK_C_GetMechanismList");
    if mechanism_list.is_null() {
        unsafe {
            *count = MECHANISM_LIST.len() as u64;
        }
        return CKR_OK;
    }
    let n = unsafe { *count } as usize;
    if n < MECHANISM_LIST.len() {
        return CKR_BUFFER_TOO_SMALL;
    }

    if let Some(max_idx) = [n, MECHANISM_LIST.len()].iter().min() {
        for (i, &mechanism_type) in (0..*max_idx).zip(MECHANISM_LIST) {
            unsafe {
                *(mechanism_list.offset(i as isize)) = mechanism_type;
            }
        }
    }

    CKR_OK
}

pub extern "C" fn CK_C_GetMechanismInfo(slotID: CK_SLOT_ID,
                                        type_: CK_MECHANISM_TYPE,
                                        info: *mut _CK_MECHANISM_INFO)
                                        -> CK_RV {
    notice!("CK_C_GetMechanismInfo");
    match type_ {
        CKM_RSA_PKCS => {
            notice!("CKM_RSA_PKCS");
            unsafe {
                *info = CK_MECHANISM_INFO {
                    ulMinKeySize: 2048,
                    ulMaxKeySize: 4096,
                    flags: CKF_SIGN | CKF_HW,
                };
            }
        }
        _ => {
            notice!("unsupported mechanism type: {}", type_);
        }
    }
    CKR_OK
}

lazy_static! {
    static ref next_session_handle : AtomicUsize = AtomicUsize::new(1);
}

pub extern "C" fn CK_C_OpenSession(slotID: CK_SLOT_ID,
                                   flags: CK_FLAGS,
                                   application: *mut ::std::os::raw::c_void,
                                   notify: CK_NOTIFY,
                                   session: *mut CK_SESSION_HANDLE)
                                   -> CK_RV {
    notice!("CK_C_OpenSession");
    if flags & CKF_SERIAL_SESSION == 0 {
        error!("CKF_SERIAL_SESSION not set");
        return CKR_SESSION_PARALLEL_NOT_SUPPORTED;
    }
    unsafe {
        *session = next_session_handle.fetch_add(1usize, SeqCst) as u64;
    }
    CKR_OK
}

pub extern "C" fn CK_C_GetSessionInfo(session: CK_SESSION_HANDLE,
                                      info: *mut _CK_SESSION_INFO)
                                      -> CK_RV {
    notice!("CK_C_GetSessionInfo");
    unsafe {
        *info = CK_SESSION_INFO {
            slotID: 0,
            state: CKS_RW_USER_FUNCTIONS,
            flags: CKF_RW_SESSION | CKF_SERIAL_SESSION,
            ulDeviceError: 0,
        };
    }
    CKR_OK
}

pub extern "C" fn CK_C_FindObjectsInit(session: CK_SESSION_HANDLE,
                                       templ: *mut _CK_ATTRIBUTE,
                                       count: ::std::os::raw::c_ulong)
                                       -> CK_RV {
    notice!("CK_C_FindObjectsInit");
    CKR_OK
}

pub extern "C" fn CK_C_FindObjects(session: CK_SESSION_HANDLE,
                                   object: *mut CK_OBJECT_HANDLE,
                                   max_object_count: ::std::os::raw::c_ulong,
                                   object_count: *mut ::std::os::raw::c_ulong)
                                   -> CK_RV {
    notice!("CK_C_FindObjects");
    unsafe {
        *object_count = 0;
    }
    CKR_OK
}

pub extern "C" fn CK_C_FindObjectsFinal(session: CK_SESSION_HANDLE) -> CK_RV {
    notice!("CK_C_FindObjectsFinal");
    CKR_OK
}

pub extern "C" fn CK_C_GetAttributeValue(session: CK_SESSION_HANDLE,
                                         object: CK_OBJECT_HANDLE,
                                         templ: *mut _CK_ATTRIBUTE,
                                         count: ::std::os::raw::c_ulong)
                                         -> CK_RV {
    notice!("CK_C_GetAttributeValue");
    CKR_FUNCTION_NOT_SUPPORTED
}

pub extern "C" fn CK_C_Finalize(pReserved: *mut ::std::os::raw::c_void) -> CK_RV {
    notice!("CK_C_Finalize");
    match OLD_STDERR_FD.lock() {
        Ok(old_stderr_fd) => {
            match *old_stderr_fd {
                Some(fd) => {
                    unsafe { libc::dup2(fd, libc::STDERR_FILENO) };
                },
                _ => {},
            };
        },
        Err(_) => { },
    };
    CKR_OK
}

static mut FUNCTION_LIST: _CK_FUNCTION_LIST = _CK_FUNCTION_LIST {
    version: _CK_VERSION {
        major: 1,
        minor: 0,
    },
    C_Initialize: CK_C_Initialize,
    C_Finalize: CK_C_Finalize,
    C_GetInfo: CK_C_GetInfo,
    C_GetFunctionList: CK_C_GetFunctionList,
    C_GetSlotList: CK_C_GetSlotList,
    C_GetSlotInfo: CK_C_GetSlotInfo,
    C_GetTokenInfo: CK_C_GetTokenInfo,
    C_GetMechanismList: CK_C_GetMechanismList,
    C_GetMechanismInfo: CK_C_GetMechanismInfo,
    C_InitToken: CK_C_InitToken,
    C_InitPIN: CK_C_InitPIN,
    C_SetPIN: CK_C_SetPIN,
    C_OpenSession: CK_C_OpenSession,
    C_CloseSession: CK_C_CloseSession,
    C_CloseAllSessions: CK_C_CloseAllSessions,
    C_GetSessionInfo: CK_C_GetSessionInfo,
    C_GetOperationState: CK_C_GetOperationState,
    C_SetOperationState: CK_C_SetOperationState,
    C_Login: CK_C_Login,
    C_Logout: CK_C_Logout,
    C_CreateObject: CK_C_CreateObject,
    C_CopyObject: CK_C_CopyObject,
    C_DestroyObject: CK_C_DestroyObject,
    C_GetObjectSize: CK_C_GetObjectSize,
    C_GetAttributeValue: CK_C_GetAttributeValue,
    C_SetAttributeValue: CK_C_SetAttributeValue,
    C_FindObjectsInit: CK_C_FindObjectsInit,
    C_FindObjects: CK_C_FindObjects,
    C_FindObjectsFinal: CK_C_FindObjectsFinal,
    C_EncryptInit: CK_C_EncryptInit,
    C_Encrypt: CK_C_Encrypt,
    C_EncryptUpdate: CK_C_EncryptUpdate,
    C_EncryptFinal: CK_C_EncryptFinal,
    C_DecryptInit: CK_C_DecryptInit,
    C_Decrypt: CK_C_Decrypt,
    C_DecryptUpdate: CK_C_DecryptUpdate,
    C_DecryptFinal: CK_C_DecryptFinal,
    C_DigestInit: CK_C_DigestInit,
    C_Digest: CK_C_Digest,
    C_DigestUpdate: CK_C_DigestUpdate,
    C_DigestKey: CK_C_DigestKey,
    C_DigestFinal: CK_C_DigestFinal,
    C_SignInit: CK_C_SignInit,
    C_Sign: CK_C_Sign,
    C_SignUpdate: CK_C_SignUpdate,
    C_SignFinal: CK_C_SignFinal,
    C_SignRecoverInit: CK_C_SignRecoverInit,
    C_SignRecover: CK_C_SignRecover,
    C_VerifyInit: CK_C_VerifyInit,
    C_Verify: CK_C_Verify,
    C_VerifyUpdate: CK_C_VerifyUpdate,
    C_VerifyFinal: CK_C_VerifyFinal,
    C_VerifyRecoverInit: CK_C_VerifyRecoverInit,
    C_VerifyRecover: CK_C_VerifyRecover,
    C_DigestEncryptUpdate: CK_C_DigestEncryptUpdate,
    C_DecryptDigestUpdate: CK_C_DecryptDigestUpdate,
    C_SignEncryptUpdate: CK_C_SignEncryptUpdate,
    C_DecryptVerifyUpdate: CK_C_DecryptVerifyUpdate,
    C_GenerateKey: CK_C_GenerateKey,
    C_GenerateKeyPair: CK_C_GenerateKeyPair,
    C_WrapKey: CK_C_WrapKey,
    C_UnwrapKey: CK_C_UnwrapKey,
    C_DeriveKey: CK_C_DeriveKey,
    C_SeedRandom: CK_C_SeedRandom,
    C_GenerateRandom: CK_C_GenerateRandom,
    C_GetFunctionStatus: CK_C_GetFunctionStatus,
    C_CancelFunction: CK_C_CancelFunction,
    C_WaitForSlotEvent: CK_C_WaitForSlotEvent,
};
