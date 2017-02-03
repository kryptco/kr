use std::io::prelude::*;
use std::io::SeekFrom;
use std::io::BufReader;
use std::{thread, time};
use std::collections::HashSet;
use std::env;
use std::fs::File;
use std::os::unix::io::FromRawFd;
use std::sync::atomic::{AtomicBool, Ordering};

extern crate libc;
#[macro_use]
extern crate lazy_static;

lazy_static! {
    static ref HAS_RECEIVED_STDOUT : AtomicBool = AtomicBool::new(false);
}

//  Notifications from other Kryptonite SSH processes will also be logged by this process. To
//  remedy this, we stop logging once any stdout has been sent by SSH (indicating successful
//  login).
fn start_stdout_detection() {
    let mut pipe_fds : [libc::c_int ; 2] = [0, 0];
    unsafe {
        if 0 != libc::pipe(pipe_fds.as_mut_ptr()) {
            return;
        }
    }
    let read_fd : libc::c_int = pipe_fds[0];
    let write_fd : libc::c_int = pipe_fds[1];

    let mut real_stdout = unsafe { File::from_raw_fd(libc::dup(libc::STDOUT_FILENO)) };
    unsafe { libc::dup2(write_fd, libc::STDOUT_FILENO) };
    let mut pipe_read = BufReader::new( unsafe { File::from_raw_fd(read_fd) } );

    thread::spawn(move || {
        let mut byte_buf = [0u8; 1];
        if let Ok(_) = pipe_read.read_exact(&mut byte_buf) {
            HAS_RECEIVED_STDOUT.store(true, Ordering::SeqCst);
            real_stdout.write(&byte_buf);

            let mut larger_buf = [0u8; 1 << 15];
            loop {
                if let Ok(n) = pipe_read.read(&mut larger_buf) {
                    real_stdout.write(&larger_buf.split_at(n).0);
                }
            }
        }
    });
}

#[no_mangle]
pub extern "C" fn Init() {
    match env::var("KR_NO_STDERR") {
        Ok(val) => return,
        Err(e) =>{},
    };

    let home_dir = match env::home_dir() {
        Some(path) => path,
        None => return,
    };

    start_stdout_detection();

    thread::spawn(move || {
        use std::fs::OpenOptions;
        use std::env;

        let mut file = match OpenOptions::new()
            .create(true)
            .truncate(true)
            .read(true)
            .write(true)
            .open(home_dir.join(".kr/krd-notify.log")) {
                Ok(file) => file,
                Err(e) => {
                    write!(&mut std::io::stderr(), "error opening Kryptonite log file: {:?}", e);
                    return;
                },
        };
        file.seek(SeekFrom::End(0));
        let mut reader = BufReader::new(file);

        let mut printed_messages = HashSet::<String>::new();
        loop {
            let mut buf = String::new();
            match reader.read_line(&mut buf) {
                Ok(_) => {
                    if HAS_RECEIVED_STDOUT.load(Ordering::SeqCst) {
                        return;
                    }
                    if buf.len() > 1 && !printed_messages.contains(&buf) {
                        printed_messages.insert(buf.clone());
                        write!(&mut std::io::stderr(), "{}", buf);
                    } else {
                        thread::sleep(time::Duration::from_millis(10));
                    }
                },
                Err(e) => {
                    writeln!(&mut std::io::stderr(), "err: {:?}", e);
                    thread::sleep(time::Duration::from_millis(250));
                },
            };
        }
    });
}

#[cfg(test)]
mod tests {
    #[test]
    fn it_works() {
    }
}
