package types

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
