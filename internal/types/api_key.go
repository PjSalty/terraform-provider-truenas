package types

// APIKey represents an API key in TrueNAS.
type APIKey struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Key      string `json:"key,omitempty"`
	Local    bool   `json:"local"`
	Revoked  bool   `json:"revoked"`
}

// APIKeyCreateRequest represents the request body for creating an API key.
type APIKeyCreateRequest struct {
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
}

// APIKeyUpdateRequest represents the request body for updating an API key.
type APIKeyUpdateRequest struct {
	Name  string `json:"name,omitempty"`
	Reset bool   `json:"reset,omitempty"`
}
