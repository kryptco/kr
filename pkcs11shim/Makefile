all:
	cargo build --release
	cp target/release/libkrpkcs11.dylib target/release/kr-pkcs11.so || true
	cp target/release/libkrpkcs11.so target/release/kr-pkcs11.so || true
