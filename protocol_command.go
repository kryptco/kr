package kr

type CreateTeamRequest struct {
	Name string `json:"name"`
}

type CreateTeamResponse struct {
	KeyAndTeamCheckpoint *KeyAndTeamCheckpoint `json:"key_and_team_checkpoint,omitempty"`
	Error                *string               `json:"error,omitempty"`
}

type AdminKeyRequest struct{}

type AdminKeyResponse struct {
	KeyAndTeamCheckpoint *KeyAndTeamCheckpoint `json:"key_and_team_checkpoint,omitempty"`
	Error                *string               `json:"error,omitempty"`
}

type KeyAndTeamCheckpoint struct {
	PrivateKeySeed []byte `json:"seed"`
	TeamPublicKey  []byte `json:"team_public_key"`
	LastBlockHash  []byte `json:"last_block_hash"`
}
