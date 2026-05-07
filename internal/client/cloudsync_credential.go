package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// CloudSyncCredential, CloudSyncCredentialCreateRequest,
// CloudSyncCredentialUpdateRequest moved to
// internal/types/cloudsync_credential.go in the v2.0
// transport-migration prep.
type (
	CloudSyncCredential              = types.CloudSyncCredential
	CloudSyncCredentialCreateRequest = types.CloudSyncCredentialCreateRequest
	CloudSyncCredentialUpdateRequest = types.CloudSyncCredentialUpdateRequest
)

// GetCloudSyncCredential retrieves a cloud sync credential by ID.
func (c *Client) GetCloudSyncCredential(ctx context.Context, id int) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "GetCloudSyncCredential start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cloudsync/credentials/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cloud sync credential %d: %w", id, err)
	}

	var cred types.CloudSyncCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudSyncCredential success")
	return &cred, nil
}

// ListCloudSyncCredentials retrieves all cloud sync credentials.
func (c *Client) ListCloudSyncCredentials(ctx context.Context) ([]types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "ListCloudSyncCredentials start")

	resp, err := c.Get(ctx, "/cloudsync/credentials")
	if err != nil {
		return nil, fmt.Errorf("listing cloud sync credentials: %w", err)
	}

	var creds []types.CloudSyncCredential
	if err := json.Unmarshal(resp, &creds); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credentials list: %w", err)
	}

	tflog.Trace(ctx, "ListCloudSyncCredentials success")
	return creds, nil
}

// GetCloudSyncCredentialByName finds a cloud sync credential by name.
func (c *Client) GetCloudSyncCredentialByName(ctx context.Context, name string) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "GetCloudSyncCredentialByName start")

	creds, err := c.ListCloudSyncCredentials(ctx)
	if err != nil {
		return nil, err
	}

	for i := range creds {
		if creds[i].Name == name {
			return &creds[i], nil
		}
	}

	tflog.Trace(ctx, "GetCloudSyncCredentialByName success")
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("cloud sync credential %q not found", name),
	}
}

// CreateCloudSyncCredential creates a new cloud sync credential.
func (c *Client) CreateCloudSyncCredential(ctx context.Context, req *types.CloudSyncCredentialCreateRequest) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "CreateCloudSyncCredential start")

	resp, err := c.Post(ctx, "/cloudsync/credentials", req)
	if err != nil {
		return nil, fmt.Errorf("creating cloud sync credential %q: %w", req.Name, err)
	}

	var cred types.CloudSyncCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudSyncCredential success")
	return &cred, nil
}

// UpdateCloudSyncCredential updates an existing cloud sync credential.
func (c *Client) UpdateCloudSyncCredential(ctx context.Context, id int, req *types.CloudSyncCredentialUpdateRequest) (*types.CloudSyncCredential, error) {
	tflog.Trace(ctx, "UpdateCloudSyncCredential start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cloudsync/credentials/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cloud sync credential %d: %w", id, err)
	}

	var cred types.CloudSyncCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCloudSyncCredential success")
	return &cred, nil
}

// DeleteCloudSyncCredential deletes a cloud sync credential.
func (c *Client) DeleteCloudSyncCredential(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCloudSyncCredential start")

	_, err := c.Delete(ctx, fmt.Sprintf("/cloudsync/credentials/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting cloud sync credential %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteCloudSyncCredential success")
	return nil
}
