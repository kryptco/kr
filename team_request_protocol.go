package kr

type CreateTeamRequest struct {
	TeamInfo TeamInfo `json:"team_info"`
}

type TeamCheckpoint struct {
	PublicKey     []byte `json:"public_key"`
	TeamPublicKey []byte `json:"team_public_key"`
	LastBlockHash []byte `json:"last_block_hash"`
}
type TeamOperationRequest struct {
	Operation RequestableTeamOperation `json:"operation"`
}

type TeamOperationResponse struct {
	PostedBlockHash []byte                     `json:"posted_block_hash"`
	Data            *TeamOperationResponseData `json:"data,omitempty"`
}

type TeamOperationResponseData struct {
	InviteLink *string `json:"invite_link,omitempty"`
}

type ReadTeamRequest struct {
	PublicKey []byte `json:"public_key"`
}

type ReadTeamResponse struct {
	SignerPublicKey []byte `json:"signer_public_key"`
	Token           string `json:"token"`
	Signature       []byte `json:"signature"`
}

type ReadToken struct {
	Time *TimeToken `json:"time,omitempty"`
}

type TimeToken struct {
	PublicKey  []byte `json:"public_key"`
	Expiration uint64 `json:"expiration"`
}

type RequestableTeamOperation struct {
	Invite       *struct{} `json:"invite,omitempty"`
	CancelInvite *struct{} `json:"cancel_invite,omitempty"`

	RemoveMember *[]byte `json:"remove_member,omitempty"`

	SetPolicy   *Policy   `json:"set_policy,omitempty"`
	SetTeamInfo *TeamInfo `json:"set_team_info,omitempty"`

	PinHostKey   *SSHHostKey `json:"pin_host_key,omitempty"`
	UnpinHostKey *SSHHostKey `json:"unpin_host_key,omitempty"`

	AddLoggingEndpoint    *LoggingEndpoint `json:"add_logging_endpoint,omitempty"`
	RemoveLoggingEndpoint *LoggingEndpoint `json:"remove_logging_endpoint,omitempty"`

	AddAdmin    *[]byte `json:"add_admin,omitempty"`
	RemoveAdmin *[]byte `json:"remove_admin,omitempty"`
}

type LogDecryptionRequest struct {
	WrappedKey WrappedKey `json:"wrapped_key"`
}

type LogDecryptionResponse struct {
	LogDecryptionKey []byte `json:"log_decryption_key"`
}

type Policy struct {
	TemporaryApprovalSeconds *uint64 `json:"temporary_approval_seconds,omitempty"`
}

type TeamInfo struct {
	Name string `json:"name,omitempty"`
}

type WrappedKey struct {
	PublicKey  []byte `json:"public_key"`
	Ciphertext []byte `json:"ciphertext"`
}

type SSHHostKey struct {
	Host      string `json:"host"`
	PublicKey []byte `json:"public_key"`
}

type LoggingEndpoint struct {
	CommandEncrypted *struct{} `json:"command_encrypted,omitempty"`
}
