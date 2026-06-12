package types

// CloudSyncCredential represents a cloud storage credential on TrueNAS SCALE.
// The `provider` field is polymorphic; its shape depends on the provider type
// (e.g. {"type": "S3", "access_key_id": "...", "secret_access_key": "..."}).
type CloudSyncCredential struct {
	ID       int                    `json:"id"`
	Name     string                 `json:"name"`
	Provider map[string]interface{} `json:"provider"`
}

// CloudSyncCredentialCreateRequest is the body for creating a cloud sync credential.
type CloudSyncCredentialCreateRequest struct {
	Name     string                 `json:"name"`
	Provider map[string]interface{} `json:"provider"`
}

// CloudSyncCredentialUpdateRequest is the body for updating a cloud sync credential.
type CloudSyncCredentialUpdateRequest struct {
	Name     string                 `json:"name,omitempty"`
	Provider map[string]interface{} `json:"provider,omitempty"`
}
