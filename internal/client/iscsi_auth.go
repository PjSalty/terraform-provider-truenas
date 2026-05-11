package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ISCSIAuth represents an iSCSI CHAP authentication credential set.
type ISCSIAuth struct {
	ID            int    `json:"id"`
	Tag           int    `json:"tag"`
	User          string `json:"user"`
	Secret        string `json:"secret"`
	Peeruser      string `json:"peeruser"`
	Peersecret    string `json:"peersecret"`
	DiscoveryAuth string `json:"discovery_auth"`
}

// ISCSIAuthCreateRequest is the create payload.
type ISCSIAuthCreateRequest struct {
	Tag           int    `json:"tag"`
	User          string `json:"user"`
	Secret        string `json:"secret"`
	Peeruser      string `json:"peeruser,omitempty"`
	Peersecret    string `json:"peersecret,omitempty"`
	DiscoveryAuth string `json:"discovery_auth,omitempty"`
}

// ISCSIAuthUpdateRequest is the update payload.
type ISCSIAuthUpdateRequest struct {
	Tag           *int    `json:"tag,omitempty"`
	User          *string `json:"user,omitempty"`
	Secret        *string `json:"secret,omitempty"`
	Peeruser      *string `json:"peeruser,omitempty"`
	Peersecret    *string `json:"peersecret,omitempty"`
	DiscoveryAuth *string `json:"discovery_auth,omitempty"`
}

// GetISCSIAuth retrieves an iSCSI auth entry by ID.
func (c *Client) GetISCSIAuth(ctx context.Context, id int) (*ISCSIAuth, error) {
	tflog.Trace(ctx, "GetISCSIAuth start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/auth/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI auth %d: %w", id, err)
	}

	var a ISCSIAuth
	if err := json.Unmarshal(resp, &a); err != nil {
		return nil, fmt.Errorf("parsing iSCSI auth response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIAuth success")
	return &a, nil
}

// CreateISCSIAuth creates an iSCSI auth entry.
func (c *Client) CreateISCSIAuth(ctx context.Context, req *ISCSIAuthCreateRequest) (*ISCSIAuth, error) {
	tflog.Trace(ctx, "CreateISCSIAuth start")

	resp, err := c.Post(ctx, "/iscsi/auth", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI auth: %w", err)
	}

	var a ISCSIAuth
	if err := json.Unmarshal(resp, &a); err != nil {
		return nil, fmt.Errorf("parsing iSCSI auth create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIAuth success")
	return &a, nil
}

// UpdateISCSIAuth updates an iSCSI auth entry by ID.
func (c *Client) UpdateISCSIAuth(ctx context.Context, id int, req *ISCSIAuthUpdateRequest) (*ISCSIAuth, error) {
	tflog.Trace(ctx, "UpdateISCSIAuth start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/auth/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI auth %d: %w", id, err)
	}

	var a ISCSIAuth
	if err := json.Unmarshal(resp, &a); err != nil {
		return nil, fmt.Errorf("parsing iSCSI auth update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIAuth success")
	return &a, nil
}

// DeleteISCSIAuth deletes an iSCSI auth entry.
func (c *Client) DeleteISCSIAuth(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIAuth start")

	_, err := c.Delete(ctx, fmt.Sprintf("/iscsi/auth/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting iSCSI auth %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteISCSIAuth success")
	return nil
}
