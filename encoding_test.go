package kr

import (
	"bytes"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/ssh"
)

var pubWire, _ = base64.StdEncoding.DecodeString("AAAAB3NzaC1yc2EAAAADAQABAAACAQCz5NxDjQgHtjnI4ilK7drJyZEzZyDCRrQgQWkKTTgKm/zH1K/3ygs63UW4zJB2sGR/UVhTJ8f11jyiRvSdEMzb47ERCxlVwA96C5i2Ha5JzxcV+ERY4uqkxIjfbsDdvdwewb3kMRYqVcPeBXWnwZ7VAkGWNZI3KP2CtSh/fsJ3xDSztzxtqZOWlPfRO4W0ClQvZpHkYRAoQH+7XHZFh1B/lw6hlSQmT5+q+WBkG1YGQUuFCyIZmnJat8YJAQkXOWBuOqxkWRQsPxd8LZrP87Ut32Lmz3oy0nxlRI1H56ebzj/vw/xpwntISg1XlsniXols75CLjs/N5DCM+KcxhE7Y49dui53/TQgc8SRYRIUy00c6Wll7QrqT5OvcGDi8kKGGWiWz1hquyT4Yb3ULWxf7sTTeVt+Ldrbxf3J3orFVaHkgI5HTduTbu45y96yPutJncX8CwoPI/l3pZ2684EXGwltHeUN1REqJwRMzaDc0A0ok3vFN5epoaBixhygWW1kK4CkzZ7UQ9XWz99ba7EVArz79tJZPLG7M4y8OIPSyoRZDcaCDBNyRIofiyAJlfi8zV7MAN6f2xjj1w8jfzp9FgX79K6DTp4tBJDIkum4YfUlKne7KHINLY2xMggdi6dkDucaEX0n1e1TrMe8CpCPzak1dDf99q3XNwVJaThZkpw==")

func TestWireRSAPubToRSAPub(t *testing.T) {
	pk, err := SSHWireRSAPublicKeyToRSAPublicKey(pubWire)
	if err != nil {
		t.Fatal(err)
	}
	sshPk, err := ssh.NewPublicKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	marshaled := sshPk.Marshal()

	if !bytes.Equal(marshaled, pubWire) {
		t.Fatal("marshaled key != pubWire")
	}
}
