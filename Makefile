OS ?= $(shell ./install/os.sh)
UNAME_S := $(shell uname -s)

ifeq ($(UNAME_S),Linux)
	PREFIX ?= /usr
	SUDO ?= sudo
endif
ifeq ($(UNAME_S),Darwin)
	PREFIX ?= /usr/local

	OSXRELEASE := $(shell uname -r | sed 's/\..*//')
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
DSTBIN = $(DESTDIR)$(PREFIX)/bin

SRCLIB = $(PWD)/lib
DSTLIB = $(DESTDIR)$(PREFIX)/lib

SRCFRAMEWORK = $(PWD)/Frameworks
DSTFRAMEWORK = $(PREFIX)/Frameworks

CONFIGURATION ?= Release
ifeq ($(CONFIGURATION), Release)
	CARGO_RELEASE = --release
	CARGO_TARGET_SCHEME = release
else ifeq ($(CONFIGURATION), Debug)
	CARGO_TARGET_SCHEME = debug
else
$(error Invalid $$CONFIGURATION)
endif

LINK_LIBSIGCHAIN_LDFLAGS = -L ${PWD}/sigchain/target/${CARGO_TARGET_SCHEME} 

all:
	-rm -rf bin lib Frameworks
	-mkdir -p bin
	-mkdir -p lib
	-mkdir -p Frameworks
	go clean -cache || true
ifeq ($(UNAME_S),Darwin)
ifeq ($(shell expr $(OSXRELEASE) \>= 16), 1)
		cd krbtle && xcodebuild -configuration $(CONFIGURATION) -archivePath $(SRCFRAMEWORK) -scheme krbtle-Package
		-rm -rf $(SRCFRAMEWORK)/krbtle.framework
		cp -R krbtle/build/$(CONFIGURATION)/krbtle.framework $(SRCFRAMEWORK)/krbtle.framework
endif
endif
	cd sigchain && CARGO_RELEASE=$(CARGO_RELEASE) make libsigchain-with-dashboard
	cd kr; CGO_LDFLAGS="$(LINK_LIBSIGCHAIN_LDFLAGS)" go build -ldflags="-s -w" $(GO_TAGS) -o ../bin/kr
	cd krd/main; CGO_LDFLAGS="$(CGO_LDFLAGS) $(LINK_LIBSIGCHAIN_LDFLAGS)" go build -ldflags="-s -w" $(GO_TAGS) -o ../../bin/krd
	cd pkcs11shim; make; cp target/release/kr-pkcs11.so ../lib/
	cd krssh; go build -ldflags="-s -w" $(GO_TAGS) -o ../bin/krssh
	cd krgpg; go build $(GO_TAGS) -ldflags="-s -w" -o ../bin/krgpg

clean:
	rm -rf bin/

check:
	go clean -cache || true
	go test $(GO_TAGS) github.com/kryptco/kr
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS) $(LINK_LIBSIGCHAIN_LDFLAGS)" go test $(GO_TAGS) github.com/kryptco/kr/krd github.com/kryptco/kr/krd/main github.com/kryptco/kr/krdclient github.com/kryptco/kr/kr github.com/kryptco/kr/krssh github.com/kryptco/kr/krgpg
	cd pkcs11shim; cargo test
	cd sigchain; CARGO_RELEASE=$(CARGO_RELEASE) make check-libsigchain-with-dashboard

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
	pkill -U $(USER) -x krd
	kr uninstall
	$(SUDO) rm -f $(DSTBIN)/kr
	$(SUDO) rm -f $(DSTBIN)/krd
	$(SUDO) rm -f $(DSTBIN)/krssh
	$(SUDO) rm -f $(DSTBIN)/krgpg
	$(SUDO) rm -f $(DSTLIB)/kr-pkcs11.so
