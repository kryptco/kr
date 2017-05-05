//! A cache for users and groups provided by the OS.
//!
//! ## Caching, multiple threads, and mutability
//!
//! The `UsersCache` type is caught between a rock and a hard place when it comes
//! to providing references to users and groups.
//!
//! Instead of returning a fresh `User` struct each time, for example, it will
//! return a reference to the version it currently has in its cache. So you can
//! ask for User #501 twice, and you’ll get a reference to the same value both
//! time. Its methods are *idempotent* -- calling one multiple times has the
//! same effect as calling one once.
//!
//! This works fine in theory, but in practice, the cache has to update its own
//! state somehow: it contains several `HashMap`s that hold the result of user
//! and group lookups. Rust provides mutability in two ways:
//!
//! 1. Have its methods take `&mut self`, instead of `&self`, allowing the
//!   internal maps to be mutated (“inherited mutability”)
//! 2. Wrap the internal maps in a `RefCell`, allowing them to be modified
//!   (“interior mutability”).
//!
//! Unfortunately, Rust is also very protective of references to a mutable
//! value. In this case, switching to `&mut self` would only allow for one user
//! to be read at a time!
//!
//! ``norun
//! let mut cache = UsersCache::empty_cache();
//! let uid   = cache.get_current_uid();                     // OK...
//! let user  = cache.get_user_by_uid(uid).unwrap()          // OK...
//! let group = cache.get_group_by_gid(user.primary_group);  // No!
//! ```
//!
//! When we get the `user`, it returns an optional reference (which we unwrap)
//! to the user’s entry in the cache. This is a reference to something contained
//! in a mutable value. Then, when we want to get the user’s primary group, it
//! will return *another* reference to the same mutable value. This is something
//! that Rust explicitly disallows!
//!
//! The compiler wasn’t on our side with Option 1, so let’s try Option 2:
//! changing the methods back to `&self` instead of `&mut self`, and using
//! `RefCell`s internally. However, Rust is smarter than this, and knows that
//! we’re just trying the same trick as earlier. A simplified implementation of
//! a user cache lookup would look something like this:
//!
//! ``norun
//! fn get_user_by_uid(&self, uid: uid_t) -> Option<&User> {
//!     let users = self.users.borrow_mut();
//!     users.get(uid)
//! }
//! ```
//!
//! Rust won’t allow us to return a reference like this because the `Ref` of the
//! `RefCell` just gets dropped at the end of the method, meaning that our
//! reference does not live long enough.
//!
//! So instead of doing any of that, we use `Arc` everywhere in order to get
//! around all the lifetime restrictions. Returning reference-counted users and
//! groups mean that we don’t have to worry about further uses of the cache, as
//! the values themselves don’t count as being stored *in* the cache anymore. So
//! it can be queried multiple times or go out of scope and the values it
//! produces are not affected.

use libc::{uid_t, gid_t};
use std::borrow::ToOwned;
use std::cell::{Cell, RefCell};
use std::collections::hash_map::Entry::{Occupied, Vacant};
use std::collections::HashMap;
use std::sync::Arc;

use base::{User, Group, AllUsers};
use traits::{Users, Groups};


/// A producer of user and group instances that caches every result.
pub struct UsersCache {
    users:  BiMap<uid_t, User>,
    groups: BiMap<gid_t, Group>,

    uid:  Cell<Option<uid_t>>,
    gid:  Cell<Option<gid_t>>,
    euid: Cell<Option<uid_t>>,
    egid: Cell<Option<gid_t>>,
}

/// A kinda-bi-directional HashMap that associates keys to values, and then
/// strings back to keys. It doesn’t go the full route and offer
/// *values*-to-keys lookup, because we only want to search based on
/// usernames and group names. There wouldn’t be much point offering a “User
/// to uid” map, as the uid is present in the user struct!
struct BiMap<K, V> {
    forward:  RefCell< HashMap<K, Option<Arc<V>>> >,
    backward: RefCell< HashMap<Arc<String>, Option<K>> >,
}

// Default has to be impl'd manually here, because there's no
// Default impl on User or Group, even though those types aren't
// needed to produce a default instance of any HashMaps...

impl Default for UsersCache {
    fn default() -> UsersCache {
        UsersCache {
            users: BiMap {
                forward:  RefCell::new(HashMap::new()),
                backward: RefCell::new(HashMap::new()),
            },

            groups: BiMap {
                forward:  RefCell::new(HashMap::new()),
                backward: RefCell::new(HashMap::new()),
            },

            uid:  Cell::new(None),
            gid:  Cell::new(None),
            euid: Cell::new(None),
            egid: Cell::new(None),
        }
    }
}

impl UsersCache {

    /// Creates a new empty cache.
    pub fn new() -> UsersCache {
        UsersCache::default()
    }

    /// Creates a new cache that contains all the users present on the system.
    ///
    /// This is `unsafe` because we cannot prevent data races if two caches
    /// were attempted to be initialised on different threads at the same time.
    pub unsafe fn with_all_users() -> UsersCache {
        let cache = UsersCache::new();

        for user in AllUsers::new() {
            let uid = user.uid();
            let user_arc = Arc::new(user);
            cache.users.forward.borrow_mut().insert(uid, Some(user_arc.clone()));
            cache.users.backward.borrow_mut().insert(user_arc.name_arc.clone(), Some(uid));
        }

        cache
    }
}

impl Users for UsersCache {
    fn get_user_by_uid(&self, uid: uid_t) -> Option<Arc<User>> {
        let mut users_forward = self.users.forward.borrow_mut();

        match users_forward.entry(uid) {
            Vacant(entry) => {
                match super::get_user_by_uid(uid) {
                    Some(user) => {
                        let newsername = user.name_arc.clone();
                        let mut users_backward = self.users.backward.borrow_mut();
                        users_backward.insert(newsername, Some(uid));

                        let user_arc = Arc::new(user);
                        entry.insert(Some(user_arc.clone()));
                        Some(user_arc)
                    },
                    None => {
                        entry.insert(None);
                        None
                    }
                }
            },
            Occupied(entry) => entry.get().clone(),
        }
    }

    fn get_user_by_name(&self, username: &str) -> Option<Arc<User>> {
        let mut users_backward = self.users.backward.borrow_mut();

        // to_owned() could change here:
        // https://github.com/rust-lang/rfcs/blob/master/text/0509-collections-reform-part-2.md#alternatives-to-toowned-on-entries
        match users_backward.entry(Arc::new(username.to_owned())) {
            Vacant(entry) => {
                match super::get_user_by_name(username) {
                    Some(user) => {
                        let uid = user.uid();
                        let user_arc = Arc::new(user);

                        let mut users_forward = self.users.forward.borrow_mut();
                        users_forward.insert(uid, Some(user_arc.clone()));
                        entry.insert(Some(uid));

                        Some(user_arc)
                    },
                    None => {
                        entry.insert(None);
                        None
                    }
                }
            },
            Occupied(entry) => match *entry.get() {
                Some(uid) => {
                    let users_forward = self.users.forward.borrow_mut();
                    users_forward[&uid].clone()
                }
                None => None,
            }
        }
    }

    fn get_current_uid(&self) -> uid_t {
        match self.uid.get() {
            Some(uid) => uid,
            None => {
                let uid = super::get_current_uid();
                self.uid.set(Some(uid));
                uid
            }
        }
    }

    fn get_current_username(&self) -> Option<Arc<String>> {
        let uid = self.get_current_uid();
        self.get_user_by_uid(uid).map(|u| u.name_arc.clone())
    }

    fn get_effective_uid(&self) -> uid_t {
        match self.euid.get() {
            Some(uid) => uid,
            None => {
                let uid = super::get_effective_uid();
                self.euid.set(Some(uid));
                uid
            }
        }
    }

    fn get_effective_username(&self) -> Option<Arc<String>> {
        let uid = self.get_effective_uid();
        self.get_user_by_uid(uid).map(|u| u.name_arc.clone())
    }
}

impl Groups for UsersCache {
    fn get_group_by_gid(&self, gid: gid_t) -> Option<Arc<Group>> {
        let mut groups_forward = self.groups.forward.borrow_mut();

        match groups_forward.entry(gid) {
            Vacant(entry) => {
                let group = super::get_group_by_gid(gid);
                match group {
                    Some(group) => {
                        let new_group_name = group.name_arc.clone();
                        let mut groups_backward = self.groups.backward.borrow_mut();
                        groups_backward.insert(new_group_name, Some(gid));

                        let group_arc = Arc::new(group);
                        entry.insert(Some(group_arc.clone()));
                        Some(group_arc)
                    },
                    None => {
                        entry.insert(None);
                        None
                    }
                }
            },
            Occupied(entry) => entry.get().clone(),
        }
    }

    fn get_group_by_name(&self, group_name: &str) -> Option<Arc<Group>> {
        let mut groups_backward = self.groups.backward.borrow_mut();

        // to_owned() could change here:
        // https://github.com/rust-lang/rfcs/blob/master/text/0509-collections-reform-part-2.md#alternatives-to-toowned-on-entries
        match groups_backward.entry(Arc::new(group_name.to_owned())) {
            Vacant(entry) => {
                let user = super::get_group_by_name(group_name);
                match user {
                    Some(group) => {
                        let group_arc = Arc::new(group.clone());
                        let gid = group.gid();

                        let mut groups_forward = self.groups.forward.borrow_mut();
                        groups_forward.insert(gid, Some(group_arc.clone()));
                        entry.insert(Some(gid));

                        Some(group_arc)
                    },
                    None => {
                        entry.insert(None);
                        None
                    }
                }
            },
            Occupied(entry) => match *entry.get() {
                Some(gid) => {
                    let groups_forward = self.groups.forward.borrow_mut();
                    groups_forward[&gid].as_ref().cloned()
                }
                None => None,
            }
        }
    }

    fn get_current_gid(&self) -> gid_t {
        match self.gid.get() {
            Some(gid) => gid,
            None => {
                let gid = super::get_current_gid();
                self.gid.set(Some(gid));
                gid
            }
        }
    }

    fn get_current_groupname(&self) -> Option<Arc<String>> {
        let gid = self.get_current_gid();
        self.get_group_by_gid(gid).map(|g| g.name_arc.clone())
    }

    fn get_effective_gid(&self) -> gid_t {
        match self.egid.get() {
            Some(gid) => gid,
            None => {
                let gid = super::get_effective_gid();
                self.egid.set(Some(gid));
                gid
            }
        }
    }

    fn get_effective_groupname(&self) -> Option<Arc<String>> {
        let gid = self.get_effective_gid();
        self.get_group_by_gid(gid).map(|g| g.name_arc.clone())
    }
}
