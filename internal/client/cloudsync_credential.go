package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Cloud Sync Credential API ---
//
// Cloud Sync Credentials are distinct from Keychain Credentials on TrueNAS
// SCALE: they live under /cloudsync/credentials and hold provider-specific
// authentication payloads (S3 access keys, B2 application keys, Azure
// account keys, etc.) used by cloud_sync and cloud_backup tasks.

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

// GetCloudSyncCredential retrieves a cloud sync credential by ID.
func (c *Client) GetCloudSyncCredential(ctx context.Context, id int) (*CloudSyncCredential, error) {
	tflog.Trace(ctx, "GetCloudSyncCredential start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cloudsync/credentials/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cloud sync credential %d: %w", id, err)
	}

	var cred CloudSyncCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential response: %w", err)
	}

	tflog.Trace(ctx, "GetCloudSyncCredential success")
	return &cred, nil
}

// ListCloudSyncCredentials retrieves all cloud sync credentials.
func (c *Client) ListCloudSyncCredentials(ctx context.Context) ([]CloudSyncCredential, error) {
	tflog.Trace(ctx, "ListCloudSyncCredentials start")

	resp, err := c.Get(ctx, "/cloudsync/credentials")
	if err != nil {
		return nil, fmt.Errorf("listing cloud sync credentials: %w", err)
	}

	var creds []CloudSyncCredential
	if err := json.Unmarshal(resp, &creds); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credentials list: %w", err)
	}

	tflog.Trace(ctx, "ListCloudSyncCredentials success")
	return creds, nil
}

// GetCloudSyncCredentialByName finds a cloud sync credential by name.
func (c *Client) GetCloudSyncCredentialByName(ctx context.Context, name string) (*CloudSyncCredential, error) {
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
func (c *Client) CreateCloudSyncCredential(ctx context.Context, req *CloudSyncCredentialCreateRequest) (*CloudSyncCredential, error) {
	tflog.Trace(ctx, "CreateCloudSyncCredential start")

	resp, err := c.Post(ctx, "/cloudsync/credentials", req)
	if err != nil {
		return nil, fmt.Errorf("creating cloud sync credential %q: %w", req.Name, err)
	}

	var cred CloudSyncCredential
	if err := json.Unmarshal(resp, &cred); err != nil {
		return nil, fmt.Errorf("parsing cloud sync credential create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCloudSyncCredential success")
	return &cred, nil
}

// UpdateCloudSyncCredential updates an existing cloud sync credential.
func (c *Client) UpdateCloudSyncCredential(ctx context.Context, id int, req *CloudSyncCredentialUpdateRequest) (*CloudSyncCredential, error) {
	tflog.Trace(ctx, "UpdateCloudSyncCredential start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cloudsync/credentials/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cloud sync credential %d: %w", id, err)
	}

	var cred CloudSyncCredential
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
