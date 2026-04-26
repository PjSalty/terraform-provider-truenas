package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- API Key API ---

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

// GetAPIKey retrieves an API key by ID.
func (c *Client) GetAPIKey(ctx context.Context, id int) (*APIKey, error) {
	tflog.Trace(ctx, "GetAPIKey start")

	resp, err := c.Get(ctx, fmt.Sprintf("/api_key/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting API key %d: %w", id, err)
	}

	var key APIKey
	if err := json.Unmarshal(resp, &key); err != nil {
		return nil, fmt.Errorf("parsing API key response: %w", err)
	}

	tflog.Trace(ctx, "GetAPIKey success")
	return &key, nil
}

// CreateAPIKey creates a new API key. The key value is only returned on creation.
func (c *Client) CreateAPIKey(ctx context.Context, req *APIKeyCreateRequest) (*APIKey, error) {
	tflog.Trace(ctx, "CreateAPIKey start")

	resp, err := c.Post(ctx, "/api_key", req)
	if err != nil {
		return nil, fmt.Errorf("creating API key %q: %w", req.Name, err)
	}

	var key APIKey
	if err := json.Unmarshal(resp, &key); err != nil {
		return nil, fmt.Errorf("parsing API key create response: %w", err)
	}

	tflog.Trace(ctx, "CreateAPIKey success")
	return &key, nil
}

// UpdateAPIKey updates an existing API key.
func (c *Client) UpdateAPIKey(ctx context.Context, id int, req *APIKeyUpdateRequest) (*APIKey, error) {
	tflog.Trace(ctx, "UpdateAPIKey start")

	resp, err := c.Put(ctx, fmt.Sprintf("/api_key/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating API key %d: %w", id, err)
	}

	var key APIKey
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
