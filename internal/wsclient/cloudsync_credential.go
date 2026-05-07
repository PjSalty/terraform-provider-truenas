package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for cloud sync credentials:
// cloudsync.credentials.{...}.

// GetCloudSyncCredential retrieves a cloud sync credential by ID.
func (c *Client) GetCloudSyncCredential(ctx context.Context, id int) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "GetCloudSyncCredential (ws) start")

	result, err := c.Call(ctx, "cloudsync.credentials.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting cloud sync credential %d: %w", id, err)
	}

	var cred types.CloudSyncCredential
	if err := json.Unmarshal(result, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudSyncCredential (ws) success")
	return &cred, nil
}

// ListCloudSyncCredentials retrieves all cloud sync credentials.
func (c *Client) ListCloudSyncCredentials(ctx context.Context) ([]types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "ListCloudSyncCredentials (ws) start")

	result, err := c.Call(ctx, "cloudsync.credentials.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing cloud sync credentials: %w", err)
	}

	var creds []types.CloudSyncCredential
	if err := json.Unmarshal(result, &creds); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credentials list: %w", err)
	}

	tflog.Trace(ctx, "ListCloudSyncCredentials (ws) success")
	return creds, nil
}

// GetCloudSyncCredentialByName finds a cloud sync credential by name.
func (c *Client) GetCloudSyncCredentialByName(ctx context.Context, name string) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "GetCloudSyncCredentialByName (ws) start")

	creds, err := c.ListCloudSyncCredentials(ctx)
	if err != nil {
		return nil, err
	}

	for i := range creds {
		if creds[i].Name == name {
			return &creds[i], nil
		}
	}

	tflog.Trace(ctx, "GetCloudSyncCredentialByName (ws) success")
	return nil, &RPCError{
		Code:    CodeInvalidParams,
		Message: fmt.Sprintf("cloud sync credential %q not found", name),
	}
}

// CreateCloudSyncCredential creates a new cloud sync credential.
func (c *Client) CreateCloudSyncCredential(ctx context.Context, req *types.CloudSyncCredentialCreateRequest) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "CreateCloudSyncCredential (ws) start")

	result, err := c.Call(ctx, "cloudsync.credentials.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating cloud sync credential %q: %w", req.Name, err)
	}

	var cred types.CloudSyncCredential
	if err := json.Unmarshal(result, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudSyncCredential (ws) success")
	return &cred, nil
}

// UpdateCloudSyncCredential updates an existing cloud sync credential.
func (c *Client) UpdateCloudSyncCredential(ctx context.Context, id int, req *types.CloudSyncCredentialUpdateRequest) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "UpdateCloudSyncCredential (ws) start")

	result, err := c.Call(ctx, "cloudsync.credentials.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating cloud sync credential %d: %w", id, err)
	}

	var cred types.CloudSyncCredential
	if err := json.Unmarshal(result, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudSyncCredential (ws) success")
	return &cred, nil
}

// DeleteCloudSyncCredential deletes a cloud sync credential.
func (c *Client) DeleteCloudSyncCredential(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudSyncCredential (ws) start")

	if _, err := c.Call(ctx, "cloudsync.credentials.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting cloud sync credential %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteCloudSyncCredential (ws) success")
	return nil
}
