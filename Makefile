all:
	-mkdir -p bin
	cd kr; go build -o ../bin/kr
	cd krd; go build -o ../bin/krd
	cd pkcs11shim; make; cp target/release/kr-pkcs11.so ../bin/
	cd krssh/main; go build -o ../bin/krssh


check:
	go test github.com/kryptco/kr github.com/kryptco/kr/krd github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr github.com/kryptco/kr/krssh
	cd pkcs11shim; cargo test
