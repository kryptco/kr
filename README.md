# kr
__kr__ enables SSH to authenticate with a key stored in a __Kryptonite__
([iOS](https://github.com/kryptonite-ios) or
[Android](https://github.com/kryptco/kryptonite-android)) mobile app. __kr__
runs as an SSH agent, called __krd__. When a __Kryptonite__ private key
operation is needed for authentcation, __krd__ routes this request to the
paired mobile phone, where the user decides whether to allow the operation or
not. _The private key never leaves the phone._

# Supported Operating Systems
__kr__ currently supports MacOS 10.10+ and Debian Linux (with systemd and
libsodium support, i.e. Ubuntu 15.04+).

# Easy Install
`curl https://krypt.co/kr | sh`

# Build Instructions
- [Install Go 1.5+](https://golang.org/doc/install)
- [Install Rust 1.15+](https://www.rustup.rs)
```sh
make
```

# Install / Run From Source
```sh
make install
make start
```

# Security Disclosure Policy
__Kryptonite__ follows a 7-day disclosure policy. If you find a security flaw,
please send it to `disclose@krypt.co` encrypted to the PGP key with
fingerprint `B873685251A928262210E094A70D71BE0646732C` (hosted at
`pgp.mit.edu`). We ask that you delay publication of the flaw until we have
published a fix, or seven days have passed.
