use std::io::prelude::*;
use std::io::SeekFrom;
use std::io::BufReader;
use std::{thread, time};
use std::collections::HashSet;

#[no_mangle]
pub extern "C" fn Init() {
    thread::spawn(move || {
        use std::fs::OpenOptions;
        use std::env;

        let home_dir = match env::home_dir() {
            Some(path) => path,
            None => return,
        };
        let mut file = OpenOptions::new().truncate(true).read(true).write(true).open(home_dir.join(".kr/krd-notify.log")).unwrap();
        file.seek(SeekFrom::End(0));
        let mut reader = BufReader::new(file);

        let mut printed_messages = HashSet::<String>::new();

        loop {
            let mut buf = String::new();
            match reader.read_line(&mut buf) {
                Ok(_) => {
                    if buf.len() > 1 && !printed_messages.contains(&buf) {
                        printed_messages.insert(buf.clone());
                        write!(&mut std::io::stderr(), "{}", buf);
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
