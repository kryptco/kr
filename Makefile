OS ?= $(shell ./install/os.sh)
UNAME_S := $(shell uname -s)

ifeq ($(UNAME_S),Linux)
	PREFIX ?= /usr
	SUDO ?= sudo
endif
ifeq ($(UNAME_S),Darwin)
	PREFIX ?= /usr/local
	OSXRELEASE := $(shell uname -r | sed 's/\..*//')
endif
ifeq ($(UNAME_S),FreeBSD)
	PREFIX ?= /usr/local
endif

SRCBIN = $(PWD)/bin
DSTBIN = $(DESTDIR)$(PREFIX)/bin

all:
	-rm -rf bin
	-mkdir -p bin
	go clean -cache || true
	cd src; go build -ldflags="-s -w" -o ../bin/kr ./kr
	cd src; go build -ldflags="-s -w" -o ../bin/krd ./krd
	cd src; go build -ldflags="-s -w" -o ../bin/krssh ./krssh
	cd src; go build -ldflags="-s -w" -o ../bin/krgpg ./krgpg

clean:
	rm -rf bin/

check:
	go clean -cache || true
	go test ./...

install: all
	mkdir -p $(DSTBIN)
	$(SUDO) install $(SRCBIN)/kr $(DSTBIN)
	$(SUDO) install $(SRCBIN)/krd $(DSTBIN)
	$(SUDO) install $(SRCBIN)/krssh $(DSTBIN)
	$(SUDO) install $(SRCBIN)/krgpg $(DSTBIN)

start:
ifeq ($(UNAME_S),Darwin)
	mkdir -p ~/Library/LaunchAgents
	cp install/macos/share_ext/co.krypt.krd.plist ~/Library/LaunchAgents/co.krypt.krd.plist
endif
	kr restart

uninstall:
	pkill -U $(USER) -x krd
	kr uninstall
	$(SUDO) rm -f $(DSTBIN)/kr
	$(SUDO) rm -f $(DSTBIN)/krd
	$(SUDO) rm -f $(DSTBIN)/krssh
	$(SUDO) rm -f $(DSTBIN)/krgpg
