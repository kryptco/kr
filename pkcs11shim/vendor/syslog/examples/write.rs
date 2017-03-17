extern crate syslog;

use syslog::{Facility,Severity};

fn main() {
  match syslog::unix(Facility::LOG_USER) {
    Err(e)         => println!("impossible to connect to syslog: {:?}", e),
    Ok(writer) => {
      let r = writer.send(Severity::LOG_ALERT, "hello world");
      if r.is_err() {
        println!("error sending the log {}", r.err().expect("got error"));
      }
    }
  }
}
