package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for keychain credentials: keychaincredential.{...}.

// GetKeychainCredential retrieves a keychain credential by ID.
func (c *Client) GetKeychainCredential(ctx context.Context, id int) (*types.KeychainCredential, error) {
	tflog.Trace(ctx, "GetKeychainCredential (ws) start")

	result, err := c.Call(ctx, "keychaincredential.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting keychain credential %d: %w", id, err)
	}

	var cred types.KeychainCredential
	if err := json.Unmarshal(result, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential response: %w", err)
	}

	tflog.Trace(ctx, "GetKeychainCredential (ws) success")
	return &cred, nil
}

// CreateKeychainCredential creates a new keychain credential.
func (c *Client) CreateKeychainCredential(ctx context.Context, req *types.KeychainCredentialCreateRequest) (*types.KeychainCredential, error) {
	tflog.Trace(ctx, "CreateKeychainCredential (ws) start")

	result, err := c.Call(ctx, "keychaincredential.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating keychain credential %q: %w", req.Name, err)
	}

	var cred types.KeychainCredential
	if err := json.Unmarshal(result, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential create response: %w", err)
	}

	tflog.Trace(ctx, "CreateKeychainCredential (ws) success")
	return &cred, nil
}

// UpdateKeychainCredential updates an existing keychain credential.
func (c *Client) UpdateKeychainCredential(ctx context.Context, id int, req *types.KeychainCredentialUpdateRequest) (*types.KeychainCredential, error) {
	tflog.Trace(ctx, "UpdateKeychainCredential (ws) start")

	result, err := c.Call(ctx, "keychaincredential.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating keychain credential %d: %w", id, err)
	}

	var cred types.KeychainCredential
	if err := json.Unmarshal(result, &cred); err != nil {
		return nil, fmt.Errorf("parsing keychain credential update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateKeychainCredential (ws) success")
	return &cred, nil
}

// DeleteKeychainCredential deletes a keychain credential.
func (c *Client) DeleteKeychainCredential(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteKeychainCredential (ws) start")

	if _, err := c.Call(ctx, "keychaincredential.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting keychain credential %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteKeychainCredential (ws) success")
	return nil
}
