use std::sync::Arc;

pub use libc::{uid_t, gid_t, c_int};

use super::{User, Group};

/// Trait for producers of users.
pub trait Users {

    /// Returns a User if one exists for the given user ID; otherwise, returns None.
    fn get_user_by_uid(&self, uid: uid_t) -> Option<Arc<User>>;

    /// Returns a User if one exists for the given username; otherwise, returns None.
    fn get_user_by_name(&self, username: &str) -> Option<Arc<User>>;

    /// Returns the user ID for the user running the process.
    fn get_current_uid(&self) -> uid_t;

    /// Returns the username of the user running the process.
    fn get_current_username(&self) -> Option<Arc<String>>;

    /// Returns the effective user id.
    fn get_effective_uid(&self) -> uid_t;

    /// Returns the effective username.
    fn get_effective_username(&self) -> Option<Arc<String>>;
}

/// Trait for producers of groups.
pub trait Groups {

    /// Returns a Group object if one exists for the given group ID; otherwise, returns None.
    fn get_group_by_gid(&self, gid: gid_t) -> Option<Arc<Group>>;

    /// Returns a Group object if one exists for the given groupname; otherwise, returns None.
    fn get_group_by_name(&self, group_name: &str) -> Option<Arc<Group>>;

    /// Returns the group ID for the user running the process.
    fn get_current_gid(&self) -> gid_t;

    /// Returns the group name of the user running the process.
    fn get_current_groupname(&self) -> Option<Arc<String>>;

    /// Returns the effective group id.
    fn get_effective_gid(&self) -> gid_t;

    /// Returns the effective group name.
    fn get_effective_groupname(&self) -> Option<Arc<String>>;
}