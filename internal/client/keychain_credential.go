package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// KeychainCredential, KeychainCredentialCreateRequest,
// KeychainCredentialUpdateRequest moved to
// internal/types/keychain_credential.go in the v2.0 transport-migration prep.
type (
	KeychainCredential              = types.KeychainCredential
	KeychainCredentialCreateRequest = types.KeychainCredentialCreateRequest
	KeychainCredentialUpdateRequest = types.KeychainCredentialUpdateRequest
)

// GetKeychainCredential retrieves a keychain credential by ID.
func (c *Client) GetKeychainCredential(ctx context.Context, id int) (*types.KeychainCredential, error) {
	tflog.Trace(ctx, "GetKeychainCredential start")

	resp, err := c.Get(ctx, fmt.Sprintf("/keychaincredential/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting keychain credential %d: %w", id, err)
	}

	var cred types.KeychainCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential response: %w", err)
	}

	tflog.Trace(ctx, "GetKeychainCredential success")
	return &cred, nil
}

// CreateKeychainCredential creates a new keychain credential.
func (c *Client) CreateKeychainCredential(ctx context.Context, req *types.KeychainCredentialCreateRequest) (*types.KeychainCredential, error) {
	tflog.Trace(ctx, "CreateKeychainCredential start")

	resp, err := c.Post(ctx, "/keychaincredential", req)
	if err != nil {
		return nil, fmt.Errorf("creating keychain credential %q: %w", req.Name, err)
	}

	var cred types.KeychainCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential create response: %w", err)
	}

	tflog.Trace(ctx, "CreateKeychainCredential success")
	return &cred, nil
}

// UpdateKeychainCredential updates an existing keychain credential.
func (c *Client) UpdateKeychainCredential(ctx context.Context, id int, req *types.KeychainCredentialUpdateRequest) (*types.KeychainCredential, error) {
	tflog.Trace(ctx, "UpdateKeychainCredential start")

	resp, err := c.Put(ctx, fmt.Sprintf("/keychaincredential/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating keychain credential %d: %w", id, err)
	}

	var cred types.KeychainCredential
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
