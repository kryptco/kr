//! Syslog
//!
//! This crate provides facilities to send log messages via syslog.
//! It supports Unix sockets for local syslog, UDP and TCP for remote servers.
//!
//! Messages can be passed directly without modification, or in RFC 3164 or RFC 5424 format
//!
//! The code is available on [Github](https://github.com/Geal/rust-syslog)
//!
//! # Example
//!
//! ```
//! extern crate syslog;
//!
//! use syslog::{Facility,Severity};
//!
//! fn main() {
//!   match syslog::unix(Facility::LOG_USER) {
//!     Err(e)         => println!("impossible to connect to syslog: {:?}", e),
//!     Ok(writer) => {
//!       let r = writer.send(Severity::LOG_ALERT, "hello world");
//!       if r.is_err() {
//!         println!("error sending the log {}", r.err().expect("got error"));
//!       }
//!     }
//!   }
//! }
//! ```
#![crate_type = "lib"]

extern crate unix_socket;
extern crate libc;
extern crate time;
extern crate log;

use std::result::Result;
use std::io::{self, Write};
use std::env;
use std::collections::HashMap;
use std::net::{SocketAddr,ToSocketAddrs,UdpSocket,TcpStream};
use std::sync::{Arc, Mutex};
use std::path::Path;

use libc::getpid;
use unix_socket::UnixDatagram;
use log::{Log,LogRecord,LogMetadata,LogLevel,SetLoggerError};

mod facility;
pub use facility::Facility;

pub type Priority = u8;

/// RFC 5424 structured data
pub type StructuredData = HashMap<String, HashMap<String, String>>;


#[allow(non_camel_case_types)]
#[derive(Copy,Clone)]
pub enum Severity {
  LOG_EMERG,
  LOG_ALERT,
  LOG_CRIT,
  LOG_ERR,
  LOG_WARNING,
  LOG_NOTICE,
  LOG_INFO,
  LOG_DEBUG
}

enum LoggerBackend {
  /// Unix socket, temp file path, log file path
  Unix(UnixDatagram),
  Udp(Box<UdpSocket>, SocketAddr),
  Tcp(Arc<Mutex<TcpStream>>)
}

/// Main logging structure
pub struct Logger {
  facility: Facility,
  hostname: Option<String>,
  process:  String,
  pid:      i32,
  s:        LoggerBackend
}

/// Returns a Logger using unix socket to target local syslog ( using /dev/log or /var/run/syslog)
pub fn unix(facility: Facility) -> Result<Box<Logger>, io::Error> {
    unix_custom(facility, "/dev/log").or_else(|e| if e.kind() == io::ErrorKind::NotFound {
        unix_custom(facility, "/var/run/syslog")
    } else {
        Err(e)
    })
}

/// Returns a Logger using unix socket to target local syslog at user provided path
pub fn unix_custom<P: AsRef<Path>>(facility: Facility, path: P) -> Result<Box<Logger>, io::Error> {
    let (process_name, pid) = get_process_info().unwrap();
    let sock = try!(UnixDatagram::unbound());
    try!(sock.connect(path));
    Ok(Box::new(Logger {
        facility: facility.clone(),
        hostname: None,
        process:  process_name,
        pid:      pid,
        s:        LoggerBackend::Unix(sock),
    }))
}

/// returns a UDP logger connecting `local` and `server`
pub fn udp<T: ToSocketAddrs>(local: T, server: T, hostname:String, facility: Facility) -> Result<Box<Logger>, io::Error> {
  server.to_socket_addrs().and_then(|mut server_addr_opt| {
    server_addr_opt.next().ok_or(
      io::Error::new(
        io::ErrorKind::InvalidInput,
        "invalid server address"
      )
    )
  }).and_then(|server_addr| {
    UdpSocket::bind(local).map(|socket| {
      let (process_name, pid) = get_process_info().unwrap();
      Box::new(Logger {
        facility: facility.clone(),
        hostname: Some(hostname),
        process:  process_name,
        pid:      pid,
        s:        LoggerBackend::Udp(Box::new(socket), server_addr)
      })
    })
  })
}

/// returns a TCP logger connecting `local` and `server`
pub fn tcp<T: ToSocketAddrs>(server: T, hostname: String, facility: Facility) -> Result<Box<Logger>, io::Error> {
  TcpStream::connect(server).map(|socket| {
      let (process_name, pid) = get_process_info().unwrap();
      Box::new(Logger {
        facility: facility.clone(),
        hostname: Some(hostname),
        process:  process_name,
        pid:      pid,
        s:        LoggerBackend::Tcp(Arc::new(Mutex::new(socket)))
      })
  })
}

/// Unix socket Logger init function compatible with log crate
pub fn init_unix(facility: Facility, log_level: log::LogLevelFilter) -> Result<(), SetLoggerError> {
  log::set_logger(|max_level| {
    max_level.set(log_level);
    unix(facility).unwrap()
  })
}

/// Unix socket Logger init function compatible with log crate and user provided socket path
pub fn init_unix_custom<P: AsRef<Path>>(facility: Facility, log_level: log::LogLevelFilter, path: P) -> Result<(), SetLoggerError> {
    log::set_logger(|max_level| {
      max_level.set(log_level);
      unix_custom(facility, path).unwrap()
    })
}

/// UDP Logger init function compatible with log crate
pub fn init_udp<T: ToSocketAddrs>(local: T, server: T, hostname:String, facility: Facility, log_level: log::LogLevelFilter) -> Result<(), SetLoggerError> {
  log::set_logger(|max_level| {
    max_level.set(log_level);
    udp(local, server, hostname, facility).unwrap()
  })
}

/// TCP Logger init function compatible with log crate
pub fn init_tcp<T: ToSocketAddrs>(server: T, hostname: String, facility: Facility, log_level: log::LogLevelFilter) -> Result<(), SetLoggerError> {
  log::set_logger(|max_level| {
    max_level.set(log_level);
    tcp(server, hostname, facility).unwrap()
  })
}

/// Initializes logging subsystem for log crate
///
/// This tries to connect to syslog by following ways:
///
/// 1. Unix sockets /dev/log and /var/run/syslog (in this order)
/// 2. Tcp connection to 127.0.0.1:601
/// 3. Udp connection to 127.0.0.1:514
///
/// Note the last option usually (almost) never fails in this method. So
/// this method doesn't return error even if there is no syslog.
///
/// If `application_name` is `None` name is derived from executable name
pub fn init(facility: Facility, log_level: log::LogLevelFilter,
    application_name: Option<&str>)
    -> Result<(), SetLoggerError>
{
  let backend = unix(facility).map(|logger| logger.s)
    .or_else(|_| {
        TcpStream::connect(("127.0.0.1", 601))
        .map(|s| LoggerBackend::Tcp(Arc::new(Mutex::new(s))))
    })
    .or_else(|_| {
        let udp_addr = "127.0.0.1:514".parse().unwrap();
        UdpSocket::bind(("127.0.0.1", 0))
        .map(|s| LoggerBackend::Udp(Box::new(s), udp_addr))
    }).unwrap_or_else(|e| panic!("Syslog UDP socket creating failed: {}", e));
  let (process_name, pid) = get_process_info().unwrap();
  log::set_logger(|max_level| {
    max_level.set(log_level);
    Box::new(Logger {
        facility: facility.clone(),
        hostname: None,
        process:  application_name
            .map(|v| v.to_string())
            .unwrap_or(process_name),
        pid:      pid,
        s:        backend,
    })
  })
}

impl Logger {
  /// format a message as a RFC 3164 log message
  pub fn format_3164(&self, severity:Severity, message: &str) -> String {
    if let Some(ref hostname) = self.hostname {
        format!("<{}>{} {} {}[{}]: {}",
          self.encode_priority(severity, self.facility),
          time::now().strftime("%b %d %T").unwrap(),
          hostname, self.process, self.pid, message)
    } else {
        format!("<{}>{} {}[{}]: {}",
          self.encode_priority(severity, self.facility),
          time::now().strftime("%b %d %T").unwrap(),
          self.process, self.pid, message)
    }
  }

  /// format RFC 5424 structured data as `([id (name="value")*])*`
  pub fn format_5424_structured_data(&self, data: StructuredData) -> String {
    if data.is_empty() {
      "-".to_string()
    } else {
      let mut res = String::new();
      for (id, params) in data.iter() {
        res = res + "["+id;
        for (name,value) in params.iter() {
          res = res + " " + name + "=\"" + value + "\"";
        }
        res = res + "]";
      }

      res
    }
  }

  /// format a message as a RFC 5424 log message
  pub fn format_5424(&self, severity:Severity, message_id: i32, data: StructuredData, message: &str) -> String {
    let f =  format!("<{}> {} {} {} {} {} {} {} {}",
      self.encode_priority(severity, self.facility),
      1, // version
      time::now_utc().rfc3339(),
      self.hostname.as_ref().map(|x| &x[..]).unwrap_or("localhost"),
      self.process, self.pid, message_id,
      self.format_5424_structured_data(data), message);
    return f;
  }

  fn encode_priority(&self, severity: Severity, facility: Facility) -> Priority {
    return facility as u8 | severity as u8
  }

  /// Sends a basic log message of the format `<priority> message`
  pub fn send(&self, severity: Severity, message: &str) -> Result<usize, io::Error> {
    let formatted =  format!("<{}> {}",
      self.encode_priority(severity, self.facility.clone()),
      message).into_bytes();
    self.send_raw(&formatted[..])
  }

  /// Sends a RFC 3164 log message
  pub fn send_3164(&self, severity: Severity, message: &str) -> Result<usize, io::Error> {
    let formatted = self.format_3164(severity, message).into_bytes();
    self.send_raw(&formatted[..])
  }

  /// Sends a RFC 5424 log message
  pub fn send_5424(&self, severity: Severity, message_id: i32, data: StructuredData, message: &str) -> Result<usize, io::Error> {
    let formatted = self.format_5424(severity, message_id, data, message).into_bytes();
    self.send_raw(&formatted[..])
  }

  /// Sends a message directly, without any formatting
  pub fn send_raw(&self, message: &[u8]) -> Result<usize, io::Error> {
    match self.s {
      LoggerBackend::Unix(ref dgram) => dgram.send(&message[..]),
      LoggerBackend::Udp(ref socket, ref addr)    => socket.send_to(&message[..], addr),
      LoggerBackend::Tcp(ref socket_wrap)         => {
        let mut socket = socket_wrap.lock().unwrap();
        socket.write(&message[..])
      }
    }
  }

  pub fn emerg(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_EMERG, message)
  }

  pub fn alert(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_ALERT, message)
  }

  pub fn crit(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_CRIT, message)
  }

  pub fn err(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_ERR, message)
  }

  pub fn warning(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_WARNING, message)
  }

  pub fn notice(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_NOTICE, message)
  }

  pub fn info(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_INFO, message)
  }

  pub fn debug(&self, message: &str) -> Result<usize, io::Error> {
    self.send_3164(Severity::LOG_DEBUG, message)
  }

  pub fn process_name(&self) -> &String {
    &self.process
  }

  pub fn process_id(&self) -> i32 {
    self.pid
  }

  pub fn set_process_name(&mut self, name: String) {
    self.process = name
  }

  pub fn set_process_id(&mut self, id: i32) {
    self.pid = id
  }
}

#[allow(unused_variables,unused_must_use)]
impl Log for Logger {
  fn enabled(&self, metadata: &LogMetadata) -> bool {
    true
  }

  fn log(&self, record: &LogRecord) {
    let message = &(format!("{}", record.args()));
    match record.level() {
      LogLevel::Error => self.err(message),
      LogLevel::Warn  => self.warning(message),
      LogLevel::Info  => self.info(message),
      LogLevel::Debug => self.debug(message),
      LogLevel::Trace => self.debug(message)
    };
  }
}

fn get_process_info() -> Option<(String,i32)> {
  env::current_exe().ok().and_then(|path| {
    path.file_name().and_then(|os_name| os_name.to_str()).map(|name| name.to_string())
  }).map(|name| {
    let pid = unsafe { getpid() };
    (name, pid)
  })
}

#[test]
#[allow(unused_must_use)]
fn message() {
  use std::thread;
  use std::sync::mpsc::channel;

  let r = unix(Facility::LOG_USER);
  //let r = tcp("127.0.0.1:4242", "localhost".to_string(), Facility::LOG_USER);
  if r.is_ok() {
    let w = r.unwrap();
    let m:String = w.format_3164(Severity::LOG_ALERT, "hello");
    println!("test: {}", m);
    let r = w.send_3164(Severity::LOG_ALERT, "pouet");
    if r.is_err() {
      println!("error sending: {}", r.unwrap_err());
    }
    //assert_eq!(m, "<9> test hello".to_string());

    let data = Arc::new(w);
    let (tx, rx) = channel();
    for i in 0..3 {
      let shared = data.clone();
      let tx = tx.clone();
      thread::spawn(move || {
        //let mut logger = *shared;
        let message = &format!("sent from {}", i);
        shared.send_3164(Severity::LOG_DEBUG, message);
        tx.send(());
      });
    }

    for _ in 0..3 {
      rx.recv();
    }
  }
}

