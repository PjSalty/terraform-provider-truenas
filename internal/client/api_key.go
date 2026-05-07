package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// APIKey, APIKeyCreateRequest, APIKeyUpdateRequest moved to
// internal/types/api_key.go in the v2.0 transport-migration prep.
type (
	APIKey              = types.APIKey
	APIKeyCreateRequest = types.APIKeyCreateRequest
	APIKeyUpdateRequest = types.APIKeyUpdateRequest
)

// GetAPIKey retrieves an API key by ID.
func (c *Client) GetAPIKey(ctx context.Context, id int) (*types.APIKey, error) {
	tflog.Trace(ctx, "GetAPIKey start")

	resp, err := c.Get(ctx, fmt.Sprintf("/api_key/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting API key %d: %w", id, err)
	}

	var key types.APIKey
	if err := json.Unmarshal(resp, &key); err != nil {
		return nil, fmt.Errorf("parsing API key response: %w", err)
	}

	tflog.Trace(ctx, "GetAPIKey success")
	return &key, nil
}

// CreateAPIKey creates a new API key. The key value is only returned on creation.
func (c *Client) CreateAPIKey(ctx context.Context, req *types.APIKeyCreateRequest) (*types.APIKey, error) {
	tflog.Trace(ctx, "CreateAPIKey start")

	resp, err := c.Post(ctx, "/api_key", req)
	if err != nil {
		return nil, fmt.Errorf("creating API key %q: %w", req.Name, err)
	}

	var key types.APIKey
	if err := json.Unmarshal(resp, &key); err != nil {
		return nil, fmt.Errorf("parsing API key create response: %w", err)
	}

	tflog.Trace(ctx, "CreateAPIKey success")
	return &key, nil
}

// UpdateAPIKey updates an existing API key.
func (c *Client) UpdateAPIKey(ctx context.Context, id int, req *types.APIKeyUpdateRequest) (*types.APIKey, error) {
	tflog.Trace(ctx, "UpdateAPIKey start")

	resp, err := c.Put(ctx, fmt.Sprintf("/api_key/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating API key %d: %w", id, err)
	}

	var key types.APIKey
	if err := json.Unmarshal(resp, &key); err != nil {
		return nil, fmt.Errorf("parsing API key update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateAPIKey success")
	return &key, nil
}

// DeleteAPIKey deletes an API key.
func (c *Client) DeleteAPIKey(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteAPIKey start")

	_, err := c.Delete(ctx, fmt.Sprintf("/api_key/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting API key %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteAPIKey success")
	return nil
}
