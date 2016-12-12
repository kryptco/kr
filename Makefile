all:
	-mkdir -p bin
	cd kr; go build -o ../bin/kr
	cd krd; go build -o ../bin/krd
	cd pkcs11; make; cp kr-pkcs11.so ../bin/kr-pkcs11.so

check:
	go test github.com/kryptco/kr github.com/kryptco/kr/pkcs11 github.com/kryptco/kr/krd github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr
