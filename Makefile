OS ?= $(shell ./install/os.sh)
UNAME_S := $(shell uname -s)

ifeq ($(UNAME_S),Linux)
	PREFIX ?= /usr
	SUDO = sudo
endif
ifeq ($(UNAME_S),Darwin)
	PREFIX ?= /usr/local

	OSXRELEASE := $(shell uname -r | sed 's/\..*//')
	ifeq ($(OSXRELEASE), 17)
		OSXVER = "High Sierra"
	endif
	ifeq ($(OSXRELEASE), 16)
		OSXVER = "Sierra"
	endif
	ifeq ($(OSXRELEASE), 15)
		OSXVER = "El Capitan"
	endif
	ifeq ($(OSXRELEASE), 14)
		OSXVER = "Yosemite"
	endif
	ifeq ($(OSXRELEASE), 13)
		OSXVER = "Maverick"
	endif
	ifeq ($(OSXRELEASE), 12)
		OSXVER = "Mountain Lion"
	endif
	ifeq ($(OSXRELEASE), 11)
		OSXVER = "Lion"
	endif
	ifeq ($(shell expr $(OSXRELEASE) \>= 16), 1)
		CGO_TEST_LDFLAGS += -F${PWD}/Frameworks -Wl,-rpath,${PWD}/Frameworks -framework krbtle 
		CGO_LDFLAGS += -F${PWD}/Frameworks -Wl,-rpath,@executable_path/../Frameworks -framework krbtle 
	else
		GO_TAGS = -tags nobluetooth
	endif
endif
ifeq ($(UNAME_S),FreeBSD)
	PREFIX ?= /usr/local
endif

SRCBIN = $(PWD)/bin
DSTBIN = $(PREFIX)/bin

SRCLIB = $(PWD)/lib
DSTLIB = $(PREFIX)/lib

SRCFRAMEWORK = $(PWD)/Frameworks
DSTFRAMEWORK = $(PREFIX)/Frameworks

CONFIGURATION ?= Release

all:
	-rm -rf bin lib Frameworks
	-mkdir -p bin
	-mkdir -p lib
	-mkdir -p Frameworks
ifeq ($(UNAME_S),Darwin)
ifeq ($(shell expr $(OSXRELEASE) \>= 16), 1)
		cd krbtle && xcodebuild -configuration $(CONFIGURATION) -archivePath $(SRCFRAMEWORK) -scheme krbtle-Package
		-rm -rf $(SRCFRAMEWORK)/krbtle.framework
		cp -R krbtle/build/$(CONFIGURATION)/krbtle.framework $(SRCFRAMEWORK)/krbtle.framework
endif
endif
	cd kr; go build $(GO_TAGS) -o ../bin/kr
	cd krd/main; CGO_LDFLAGS="$(CGO_LDFLAGS)" go build $(GO_TAGS) -o ../../bin/krd
	cd pkcs11shim; make; cp target/release/kr-pkcs11.so ../lib/
	cd krssh; CGO_LDFLAGS="$(CGO_LDFLAGS)" go build $(GO_TAGS) -o ../bin/krssh
	cd krgpg; go build $(GO_TAGS) -o ../bin/krgpg

clean:
	rm -rf bin/

check: vet
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" go test $(GO_TAGS) github.com/kryptco/kr github.com/kryptco/kr/krd github.com/kryptco/kr/krd/main github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr github.com/kryptco/kr/krssh github.com/kryptco/kr/krgpg
	cd pkcs11shim; cargo test

vet:
	go vet github.com/kryptco/kr github.com/kryptco/kr/krd github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr github.com/kryptco/kr/krssh github.com/kryptco/kr/krgpg

install: all
	mkdir -p $(DSTBIN)
	mkdir -p $(DSTLIB)
ifeq ($(UNAME_S),Darwin)
ifeq ($(shell expr $(OSXRELEASE) \>= 16), 1)
	mkdir -p $(DSTFRAMEWORK)
	-rm -rf $(DSTFRAMEWORK)/krbtle.framework
	cp -R $(SRCFRAMEWORK)/krbtle.framework $(DSTFRAMEWORK)/krbtle.framework
endif
endif
	$(SUDO) install $(SRCBIN)/kr $(DSTBIN)
	$(SUDO) install $(SRCBIN)/krd $(DSTBIN)
	$(SUDO) install $(SRCBIN)/krssh $(DSTBIN)
	$(SUDO) install $(SRCBIN)/krgpg $(DSTBIN)
	$(SUDO) install $(SRCLIB)/kr-pkcs11.so $(DSTLIB)

start:
ifeq ($(UNAME_S),Darwin)
	mkdir -p ~/Library/LaunchAgents
	cp share/co.krypt.krd.plist ~/Library/LaunchAgents/co.krypt.krd.plist
endif
	kr restart

uninstall:
	killall krd
	kr uninstall
	$(SUDO) rm -f $(DSTBIN)/kr
	$(SUDO) rm -f $(DSTBIN)/krd
	$(SUDO) rm -f $(DSTBIN)/krssh
	$(SUDO) rm -f $(DSTBIN)/krgpg
	$(SUDO) rm -f $(DSTLIB)/kr-pkcs11.so
