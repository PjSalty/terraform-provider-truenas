package types

// KeychainCredential represents an SSH keypair or credentials in TrueNAS.
type KeychainCredential struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

// KeychainCredentialCreateRequest represents the request body for creating a keychain credential.
type KeychainCredentialCreateRequest struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

// KeychainCredentialUpdateRequest represents the request body for updating a keychain credential.
type KeychainCredentialUpdateRequest struct {
	Name       string                 `json:"name,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
