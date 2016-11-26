package kr

type NoopTransport struct {
}

func (t NoopTransport) Setup(ps *PairingSecret) (err error) {
	return
}

func (t NoopTransport) PushAlert(ps *PairingSecret, alertText string, message []byte) (err error) {
	return
}
func (t NoopTransport) SendMessage(ps *PairingSecret, message []byte) (err error) {
	return
}

func (t NoopTransport) Read(ps *PairingSecret) (ciphertexts [][]byte, err error) {
	return
}
