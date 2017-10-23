GOBUILDFLAGS += -ldflags -s

OS ?= $(shell ./install/os.sh)

all:
	-mkdir -p bin
	cd kr; go build $(GOBUILDFLAGS) -o ../bin/kr
	cd krd/main; go build $(GOBUILDFLAGS) -o ../../bin/krd
	cd pkcs11shim; make; cp target/release/kr-pkcs11.so ../bin/
	cd krssh; go build $(GOBUILDFLAGS) -o ../bin/krssh
	cd krgpg; go build $(GOBUILDFLAGS) -o ../bin/krgpg

clean:
	rm -rf bin/


check:
	go test $(GOBUILDFLAGS) github.com/kryptco/kr github.com/kryptco/kr/krd github.com/kryptco/kr/krd/main github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr github.com/kryptco/kr/krssh github.com/kryptco/kr/krgpg
	cd pkcs11shim; cargo test

vet:
	go vet github.com/kryptco/kr github.com/kryptco/kr/krd github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr github.com/kryptco/kr/krssh github.com/kryptco/kr/krgpg

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	PREFIX ?= /usr
	SUDO = sudo
endif
ifeq ($(UNAME_S),Darwin)
	PREFIX ?= /usr/local
endif
ifeq ($(UNAME_S),FreeBSD)
	PREFIX ?= /usr/local
endif

SRCBIN = $(PWD)/bin
DSTBIN = $(PREFIX)/bin
DSTLIB = $(PREFIX)/lib
install: all
	$(SUDO) ln -sf $(SRCBIN)/kr $(DSTBIN)/kr
	$(SUDO) ln -sf $(SRCBIN)/krd $(DSTBIN)/krd
	$(SUDO) ln -sf $(SRCBIN)/krssh $(DSTBIN)/krssh
	$(SUDO) ln -sf $(SRCBIN)/krgpg $(DSTBIN)/krgpg
	$(SUDO) ln -sf $(SRCBIN)/kr-pkcs11.so $(DSTLIB)/kr-pkcs11.so
	mkdir -m 700 -p ~/.ssh
	touch ~/.ssh/config
	chmod 0600 ~/.ssh/config
	perl -0777 -ne '/# Added by Kryptonite\nHost \*\n\tPKCS11Provider $(subst /,\/,$(PREFIX))\/lib\/kr-pkcs11.so\n\tProxyCommand $(subst /,\/,$(PREFIX))\/bin\/krssh %h %p\n\tIdentityFile ~\/.ssh\/id_kryptonite\n\tIdentityFile ~\/.ssh\/id_ed25519\n\tIdentityFile ~\/.ssh\/id_rsa\n\tIdentityFile ~\/.ssh\/id_ecdsa\n\tIdentityFile ~\/.ssh\/id_dsa/ || exit(1)' ~/.ssh/config || printf '\n# Added by Kryptonite\nHost *\n\tPKCS11Provider $(PREFIX)/lib/kr-pkcs11.so\n\tProxyCommand $(PREFIX)/bin/krssh %%h %%p\n\tIdentityFile ~/.ssh/id_kryptonite\n\tIdentityFile ~/.ssh/id_ed25519\n\tIdentityFile ~/.ssh/id_rsa\n\tIdentityFile ~/.ssh/id_ecdsa\n\tIdentityFile ~/.ssh/id_dsa' >> ~/.ssh/config
start:
ifeq ($(UNAME_S),Darwin)
	mkdir -p ~/Library/LaunchAgents
	cp share/co.krypt.krd.plist ~/Library/LaunchAgents/co.krypt.krd.plist
endif
	kr restart

uninstall:
	pkill krd
	$(SUDO) rm -f $(DSTBIN)/kr
	$(SUDO) rm -f $(DSTBIN)/krd
	$(SUDO) rm -f $(DSTBIN)/krssh
	$(SUDO) rm -f $(DSTBIN)/krgpg
	$(SUDO) rm -f $(DSTLIB)/kr-pkcs11.so
	perl -0777 -p -i.kr.bak -e 's/\s*# Added by Kryptonite\nHost \*\n\tPKCS11Provider $(subst /,\/,$(PREFIX))\/lib\/kr-pkcs11.so\n\tProxyCommand $(subst /,\/,$(PREFIX))\/bin\/krssh %h %p\n\tIdentityFile ~\/.ssh\/id_kryptonite\n\tIdentityFile ~\/.ssh\/id_ed25519\n\tIdentityFile ~\/.ssh\/id_rsa\n\tIdentityFile ~\/.ssh\/id_ecdsa\n\tIdentityFile ~\/.ssh\/id_dsa//g' ~/.ssh/config 
	kr uninstall
