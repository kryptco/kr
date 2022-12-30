module krypt.co/kr

go 1.13

require (
	github.com/atotto/clipboard v0.1.2
	github.com/aws/aws-sdk-go v1.33.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/fatih/color v1.7.0
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc
	github.com/hashicorp/golang-lru v0.5.3
	github.com/keybase/saltpack v0.0.0-20190828020936-3f47e8e2e6ec
	github.com/kryptco/gf256 v0.0.0-20160413180133-bbd714a0764d // indirect
	github.com/kryptco/kr v0.0.0-20190610004835-07379580bfec
	github.com/kryptco/qr v0.0.0-20161221154700-eb334d7d50ea
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/satori/go.uuid v1.2.0
	github.com/urfave/cli v1.22.1
	github.com/youtube/vitess v2.1.1+incompatible
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
)

replace golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 => github.com/kryptco/go-crypto v0.0.0-20191020215841-c5850b359d8a
