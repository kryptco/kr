package kr

type HostAuth struct {
	HostKey   []byte   `json:"host_key"`
	Signature []byte   `json:"signature"`
	HostNames []string `json:"host_names"`
}
