extern crate users;
use users::{User, AllUsers};

fn main() {
    let mut users: Vec<User> = unsafe { AllUsers::new() }.collect();
    users.sort_by(|a, b| a.uid().cmp(&b.uid()));

    for user in users {
        println!("User {} has name {}", user.uid(), user.name());
    }
}
