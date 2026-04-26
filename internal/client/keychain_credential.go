package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Keychain Credential API ---

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

// GetKeychainCredential retrieves a keychain credential by ID.
func (c *Client) GetKeychainCredential(ctx context.Context, id int) (*KeychainCredential, error) {
	tflog.Trace(ctx, "GetKeychainCredential start")

	resp, err := c.Get(ctx, fmt.Sprintf("/keychaincredential/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting keychain credential %d: %w", id, err)
	}

	var cred KeychainCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential response: %w", err)
	}

	tflog.Trace(ctx, "GetKeychainCredential success")
	return &cred, nil
}

// CreateKeychainCredential creates a new keychain credential.
func (c *Client) CreateKeychainCredential(ctx context.Context, req *KeychainCredentialCreateRequest) (*KeychainCredential, error) {
	tflog.Trace(ctx, "CreateKeychainCredential start")

	resp, err := c.Post(ctx, "/keychaincredential", req)
	if err != nil {
		return nil, fmt.Errorf("creating keychain credential %q: %w", req.Name, err)
	}

	var cred KeychainCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential create response: %w", err)
	}

	tflog.Trace(ctx, "CreateKeychainCredential success")
	return &cred, nil
}

// UpdateKeychainCredential updates an existing keychain credential.
func (c *Client) UpdateKeychainCredential(ctx context.Context, id int, req *KeychainCredentialUpdateRequest) (*KeychainCredential, error) {
	tflog.Trace(ctx, "UpdateKeychainCredential start")

	resp, err := c.Put(ctx, fmt.Sprintf("/keychaincredential/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating keychain credential %d: %w", id, err)
	}

	var cred KeychainCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateKeychainCredential success")
	return &cred, nil
}

// DeleteKeychainCredential deletes a keychain credential.
func (c *Client) DeleteKeychainCredential(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteKeychainCredential start")

	_, err := c.Delete(ctx, fmt.Sprintf("/keychaincredential/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting keychain credential %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteKeychainCredential success")
	return nil
}
