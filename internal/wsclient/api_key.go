package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for API keys: api_key.{...}.

// GetAPIKey retrieves an API key by ID.
func (c *Client) GetAPIKey(ctx context.Context, id int) (*types.APIKey, error) {
	tflog.Trace(ctx, "GetAPIKey (ws) start")

	result, err := c.Call(ctx, "api_key.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting API key %d: %w", id, err)
	}

	var key types.APIKey
	if err := json.Unmarshal(result, &key); err != nil {
		return nil, fmt.Errorf("parsing API key response: %w", err)
	}

	tflog.Trace(ctx, "GetAPIKey (ws) success")
	return &key, nil
}

// CreateAPIKey creates a new API key. The key value is only returned on creation.
func (c *Client) CreateAPIKey(ctx context.Context, req *types.APIKeyCreateRequest) (*types.APIKey, error) {
	tflog.Trace(ctx, "CreateAPIKey (ws) start")

	result, err := c.Call(ctx, "api_key.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating API key %q: %w", req.Name, err)
	}

	var key types.APIKey
	if err := json.Unmarshal(result, &key); err != nil {
		return nil, fmt.Errorf("parsing API key create response: %w", err)
	}

	tflog.Trace(ctx, "CreateAPIKey (ws) success")
	return &key, nil
}

// UpdateAPIKey updates an existing API key.
func (c *Client) UpdateAPIKey(ctx context.Context, id int, req *types.APIKeyUpdateRequest) (*types.APIKey, error) {
	tflog.Trace(ctx, "UpdateAPIKey (ws) start")

	result, err := c.Call(ctx, "api_key.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating API key %d: %w", id, err)
	}

	var key types.APIKey
	if err := json.Unmarshal(result, &key); err != nil {
		return nil, fmt.Errorf("parsing API key update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateAPIKey (ws) success")
	return &key, nil
}

// DeleteAPIKey deletes an API key.
func (c *Client) DeleteAPIKey(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteAPIKey (ws) start")

	if _, err := c.Call(ctx, "api_key.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting API key %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteAPIKey (ws) success")
	return nil
}
