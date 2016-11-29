all:
	-mkdir -p bin
	cd kr; go build -o ../bin/kr
	cd krd; go build -o ../bin/krd
	cd pkcs11; make; cp kr-pkcs11.so ../bin/kr-pkcs11.so

check:
	go test github.com/agrinman/kr/{,pkcs11,krd,krdclient,kr}
