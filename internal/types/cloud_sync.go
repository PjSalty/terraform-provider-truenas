package types

import (
	"encoding/json"
	"fmt"
)

// CloudSync represents a cloud sync task in TrueNAS.
type CloudSync struct {
	ID           int                    `json:"id"`
	Description  string                 `json:"description,omitempty"`
	Path         string                 `json:"path"`
	Credentials  int                    `json:"credentials"`
	Direction    string                 `json:"direction"`
	TransferMode string                 `json:"transfer_mode"`
	Schedule     Schedule               `json:"schedule"`
	Enabled      bool                   `json:"enabled"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

// UnmarshalJSON handles TrueNAS returning credentials as either a plain
// integer (create/update responses) or a nested object {"id": N, ...}
// (get/list responses, both REST and JSON-RPC paths). The struct field
// is always stored as the integer ID.
//
// Mirrors the same fix on internal/client/cloud_sync.go's CloudSync
// (originally PR #12, Max Poelman). The shape difference is in the
// TrueNAS API itself, not the transport, JSON-RPC over WebSocket
// carries the same payload.
func (cs *CloudSync) UnmarshalJSON(data []byte) error {
	type Alias CloudSync
	aux := &struct {
		Credentials json.RawMessage `json:"credentials"`
		*Alias
	}{
		Alias: (*Alias)(cs),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.Credentials) == 0 {
		return nil
	}
	var credID int
	if err := json.Unmarshal(aux.Credentials, &credID); err == nil {
		cs.Credentials = credID
		return nil
	}
	var credObj struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(aux.Credentials, &credObj); err != nil {
		return fmt.Errorf("credentials field: %w", err)
	}
	cs.Credentials = credObj.ID
	return nil
}

// CloudSyncCreateRequest represents the request to create a cloud sync task.
type CloudSyncCreateRequest struct {
	Description  string                 `json:"description,omitempty"`
	Path         string                 `json:"path"`
	Credentials  int                    `json:"credentials"`
	Direction    string                 `json:"direction"`
	TransferMode string                 `json:"transfer_mode"`
	Schedule     Schedule               `json:"schedule,omitempty"`
	Enabled      bool                   `json:"enabled"`
	Attributes   map[string]interface{} `json:"attributes"`
}

// CloudSyncUpdateRequest represents the request to update a cloud sync task.
type CloudSyncUpdateRequest struct {
	Description  string                 `json:"description,omitempty"`
	Path         string                 `json:"path,omitempty"`
	Credentials  int                    `json:"credentials,omitempty"`
	Direction    string                 `json:"direction,omitempty"`
	TransferMode string                 `json:"transfer_mode,omitempty"`
	Schedule     *Schedule              `json:"schedule,omitempty"`
	Enabled      *bool                  `json:"enabled,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}
