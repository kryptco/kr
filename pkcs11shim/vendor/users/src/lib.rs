#![crate_name = "users"]
#![crate_type = "rlib"]
#![crate_type = "dylib"]

//! This is a library for getting information on Unix users and groups. It
//! supports getting the system users, and creating your own mock tables.
//!
//! In Unix, each user has an individual *user ID*, and each process has an
//! *effective user ID* that says which user’s permissions it is using.
//! Furthermore, users can be the members of *groups*, which also have names and
//! IDs. This functionality is exposed in libc, the C standard library, but as
//! an unsafe Rust interface. This wrapper library provides a safe interface,
//! using User and Group objects instead of low-level pointers and strings. It
//! also offers basic caching functionality.
//!
//! It does not (yet) offer *editing* functionality; the objects returned are
//! read-only.
//!
//!
//! ## Users
//!
//! The function `get_current_uid` returns a `uid_t` value representing the user
//! currently running the program, and the `get_user_by_uid` function scans the
//! users database and returns a User object with the user’s information. This
//! function returns `None` when there is no user for that ID.
//!
//! A `User` object has the following accessors:
//!
//! - **uid:** The user’s ID
//! - **name:** The user’s name
//! - **primary_group:** The ID of this user’s primary group
//!
//! Here is a complete example that prints out the current user’s name:
//!
//! ```rust
//! use users::{get_user_by_uid, get_current_uid};
//! let user = get_user_by_uid(get_current_uid()).unwrap();
//! println!("Hello, {}!", user.name());
//! ```
//!
//! This code assumes (with `unwrap()`) that the user hasn’t been deleted after
//! the program has started running. For arbitrary user IDs, this is **not** a
//! safe assumption: it’s possible to delete a user while it’s running a
//! program, or is the owner of files, or for that user to have never existed.
//! So always check the return values from `user_to_uid`!
//!
//! There is also a `get_current_username` function, as it’s such a common
//! operation that it deserves special treatment.
//!
//!
//! ## Caching
//!
//! Despite the above warning, the users and groups database rarely changes.
//! While a short program may only need to get user information once, a
//! long-running one may need to re-query the database many times, and a
//! medium-length one may get away with caching the values to save on redundant
//! system calls.
//!
//! For this reason, this crate offers a caching interface to the database,
//! which offers the same functionality while holding on to every result,
//! caching the information so it can be re-used.
//!
//! To introduce a cache, create a new `UsersCache` and call the same
//! methods on it. For example:
//!
//! ```rust
//! use users::{Users, Groups, UsersCache};
//! let mut cache = UsersCache::new();
//! let uid = cache.get_current_uid();
//! let user = cache.get_user_by_uid(uid).unwrap();
//! println!("Hello again, {}!", user.name());
//! ```
//!
//! This cache is **only additive**: it’s not possible to drop it, or erase
//! selected entries, as when the database may have been modified, it’s best to
//! start entirely afresh. So to accomplish this, just start using a new
//! `UsersCache`.
//!
//!
//! ## Groups
//!
//! Finally, it’s possible to get groups in a similar manner.
//! A `Group` has the following accessors:
//!
//! - **gid:** The group’s ID
//! - **name:** The group’s name
//!
//! And again, a complete example:
//!
//! ```no_run
//! use users::{Users, Groups, UsersCache};
//! let mut cache = UsersCache::new();
//! let group = cache.get_group_by_name("admin").expect("No such group 'admin'!");
//! println!("The '{}' group has the ID {}", group.name(), group.gid());
//! ```
//!
//!
//! ## Caveats
//!
//! You should be prepared for the users and groups tables to be completely
//! broken: IDs shouldn’t be assumed to map to actual users and groups, and
//! usernames and group names aren’t guaranteed to map either!
//!
//! Use the mocking module to create custom tables to test your code for these
//! edge cases.

#![warn(missing_copy_implementations)]
#![warn(missing_docs)]
#![warn(trivial_casts, trivial_numeric_casts)]
#![warn(unused_extern_crates, unused_qualifications)]

extern crate libc;
pub use libc::{uid_t, gid_t};

mod base;
pub use base::{User, Group, os};
pub use base::{get_user_by_uid, get_user_by_name};
pub use base::{get_group_by_gid, get_group_by_name};
pub use base::{get_current_uid, get_current_username};
pub use base::{get_effective_uid, get_effective_username};
pub use base::{get_current_gid, get_current_groupname};
pub use base::{get_effective_gid, get_effective_groupname};
pub use base::AllUsers;


pub mod cache;
pub use cache::UsersCache;

pub mod mock;

pub mod switch;

mod traits;
pub use traits::{Users, Groups};
