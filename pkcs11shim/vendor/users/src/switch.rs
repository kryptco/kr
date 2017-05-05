//! Functions for switching the running process’s user or group.

use std::io::{Error as IOError, Result as IOResult};
use libc::{uid_t, gid_t, c_int};

use base::{get_effective_uid, get_effective_gid};


extern {
    fn setuid(uid: uid_t) -> c_int;
    fn seteuid(uid: uid_t) -> c_int;

    fn setgid(gid: gid_t) -> c_int;
    fn setegid(gid: gid_t) -> c_int;

    fn setreuid(ruid: uid_t, euid: uid_t) -> c_int;
    fn setregid(rgid: gid_t, egid: gid_t) -> c_int;
}


/// Sets the **current user** for the running process to the one with the
/// given user ID. Uses `setuid` internally.
///
/// Typically, trying to switch to anyone other than the user already running
/// the process requires root privileges.
pub fn set_current_uid(uid: uid_t) -> IOResult<()> {
    match unsafe { setuid(uid) } {
         0 => Ok(()),
        -1 => Err(IOError::last_os_error()),
         n => unreachable!("setuid returned {}", n)
    }
}

/// Sets the **current group** for the running process to the one with the
/// given group ID. Uses `setgid` internally.
///
/// Typically, trying to switch to any group other than the group already
/// running the process requires root privileges.
pub fn set_current_gid(gid: gid_t) -> IOResult<()> {
    match unsafe { setgid(gid) } {
         0 => Ok(()),
        -1 => Err(IOError::last_os_error()),
         n => unreachable!("setgid returned {}", n)
    }
}

/// Sets the **effective user** for the running process to the one with the
/// given user ID. Uses `seteuid` internally.
///
/// Typically, trying to switch to anyone other than the user already running
/// the process requires root privileges.
pub fn set_effective_uid(uid: uid_t) -> IOResult<()> {
    match unsafe { seteuid(uid) } {
         0 => Ok(()),
        -1 => Err(IOError::last_os_error()),
         n => unreachable!("seteuid returned {}", n)
    }
}

/// Sets the **effective group** for the running process to the one with the
/// given group ID. Uses `setegid` internally.
///
/// Typically, trying to switch to any group other than the group already
/// running the process requires root privileges.
pub fn set_effective_gid(gid: gid_t) -> IOResult<()> {
    match unsafe { setegid(gid) } {
         0 => Ok(()),
        -1 => Err(IOError::last_os_error()),
         n => unreachable!("setegid returned {}", n)
    }
}

/// Sets both the **current user** and the **effective user** for the running
/// process to the ones with the given user IDs. Uses `setreuid` internally.
///
/// Typically, trying to switch to anyone other than the user already running
/// the process requires root privileges.
pub fn set_both_uid(ruid: uid_t, euid: uid_t) -> IOResult<()> {
    match unsafe { setreuid(ruid, euid) } {
         0 => Ok(()),
        -1 => Err(IOError::last_os_error()),
         n => unreachable!("setreuid returned {}", n)
    }
}

/// Sets both the **current group** and the **effective group** for the
/// running process to the ones with the given group IDs. Uses `setregid`
/// internally.
///
/// Typically, trying to switch to any group other than the group already
/// running the process requires root privileges.
pub fn set_both_gid(rgid: gid_t, egid: gid_t) -> IOResult<()> {
    match unsafe { setregid(rgid, egid) } {
         0 => Ok(()),
        -1 => Err(IOError::last_os_error()),
         n => unreachable!("setregid returned {}", n)
    }
}

/// Guard returned from a `switch_user_group` call.
pub struct SwitchUserGuard {
    uid: uid_t,
    gid: gid_t,
}

impl Drop for SwitchUserGuard {
    fn drop(&mut self) {
        // Panic on error here, as failing to set values back
        // is a possible security breach.
        set_effective_uid(self.uid).unwrap();
        set_effective_gid(self.gid).unwrap();
    }
}

/// Sets the **effective user** and the **effective group** for the current
/// scope.
///
/// Typically, trying to switch to any user or group other than the ones already
/// running the process requires root privileges.
///
/// **Use with care!** Possible security issues can happen, as Rust doesn't
/// guarantee running the destructor! If in doubt run `drop()` method on the
/// guard value manually!
///
/// ### Examples
///
/// ```no_run
/// use users::switch::switch_user_group;
///
/// {
///     let _guard = switch_user_group(1001, 1001);
///     // current and effective user and group ids are 1001
/// }
/// // back to the old values
/// ```
pub fn switch_user_group(uid: uid_t, gid: gid_t) -> IOResult<SwitchUserGuard> {
    let current_state = SwitchUserGuard {
        uid: get_effective_uid(),
        gid: get_effective_gid(),
    };

    try!(set_effective_uid(uid));
    try!(set_effective_gid(gid));
    Ok(current_state)
}
