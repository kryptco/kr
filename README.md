# DEPRECATED

**This project is not maintained.** Development will continue at https://github.com/akamai/akr.

### Krypton Developer Tools 
# The `kr` command line interface

__kr__ enables SSH to authenticate with a key stored in a __Krypton__
([iOS](https://github.com/kryptco/krypton-ios) or
[Android](https://github.com/kryptco/krypton-android)) mobile app. __kr__
runs as an SSH agent, called __krd__. When a __Krypton__ private key
operation is needed for authentication, __krd__ routes this request to the
paired mobile phone, where the user decides whether to allow the operation or
not. _The private key never leaves the phone._

__kr__ also enables Git Commit/Tag signing with a key stored in __Krypton__.
__kr__ includes an interface to gpg, called __krgpg__, that can talk with git
in order to pgp-sign git commits and tags. 

# Supported Operating Systems
__kr__ currently supports MacOS (10.10+) and Linux (64 Bit) (Debian, RHEL, CentOS, Fedora with `systemd`).

# Easy Install
`curl https://krypt.co/kr | sh`

# Build Dependencies / Instructions
We use go modules for easy dependency management.

- [Install Go 1.13+](https://golang.org/doc/install)

# Install / Run From Source
```sh
make install
make start
kr pair
```

# CONTRIBUTING
Check out `CONTRIBUTING.md`

# Security Disclosure Policy
__Krypton__ follows a 7-day disclosure policy. If you find a security flaw,
please send it to `disclose@krypt.co` encrypted to the PGP key with fingerprint
`B873685251A928262210E094A70D71BE0646732C` ([find the full key here](https://krypt.co/docs/security/disclosure-policy.html)). We ask that you
delay publication of the flaw until we have published a fix, or seven days have
passed.

# LICENSE
We are currently working on a new license for Krypton. For now, the code
is released under All Rights Reserved.
