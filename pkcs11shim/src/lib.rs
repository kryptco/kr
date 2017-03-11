#[warn(unused_extern_crates)]
#[macro_use]
extern crate lazy_static;
extern crate syslog;

mod pkcs11;
#[macro_use]
mod pkcs11_unused;
pub use pkcs11_unused::*;
mod utils;
mod pkcs11shim;
pub use pkcs11shim::*;

#[cfg(test)]
mod tests {
    #[test]
    fn it_works() {
    }
}
