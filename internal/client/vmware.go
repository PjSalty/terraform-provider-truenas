package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// VMware represents a VMware host registration in TrueNAS.
type VMware struct {
	ID         int    `json:"id"`
	Datastore  string `json:"datastore"`
	Filesystem string `json:"filesystem"`
	Hostname   string `json:"hostname"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// VMwareCreateRequest represents a request to create a VMware integration.
type VMwareCreateRequest struct {
	Datastore  string `json:"datastore"`
	Filesystem string `json:"filesystem"`
	Hostname   string `json:"hostname"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// VMwareUpdateRequest represents a request to update a VMware integration.
type VMwareUpdateRequest struct {
	Datastore  *string `json:"datastore,omitempty"`
	Filesystem *string `json:"filesystem,omitempty"`
	Hostname   *string `json:"hostname,omitempty"`
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
}

// GetVMware retrieves a VMware integration by ID.
func (c *Client) GetVMware(ctx context.Context, id int) (*VMware, error) {
	tflog.Trace(ctx, "GetVMware start")

	resp, err := c.Get(ctx, fmt.Sprintf("/vmware/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting VMware %d: %w", id, err)
	}

	var v VMware
	if err := json.Unmarshal(resp, &v); err != nil {
		return nil, fmt.Errorf("parsing VMware response: %w", err)
	}

	tflog.Trace(ctx, "GetVMware success")
	return &v, nil
}

// CreateVMware creates a new VMware integration.
func (c *Client) CreateVMware(ctx context.Context, req *VMwareCreateRequest) (*VMware, error) {
	tflog.Trace(ctx, "CreateVMware start")

	resp, err := c.Post(ctx, "/vmware", req)
	if err != nil {
		return nil, fmt.Errorf("creating VMware: %w", err)
	}

	var v VMware
	if err := json.Unmarshal(resp, &v); err != nil {
		return nil, fmt.Errorf("parsing VMware create response: %w", err)
	}

	tflog.Trace(ctx, "CreateVMware success")
	return &v, nil
}

// UpdateVMware updates an existing VMware integration.
func (c *Client) UpdateVMware(ctx context.Context, id int, req *VMwareUpdateRequest) (*VMware, error) {
	tflog.Trace(ctx, "UpdateVMware start")

	resp, err := c.Put(ctx, fmt.Sprintf("/vmware/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating VMware %d: %w", id, err)
	}

	var v VMware
	if err := json.Unmarshal(resp, &v); err != nil {
		return nil, fmt.Errorf("parsing VMware update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateVMware success")
	return &v, nil
}

// DeleteVMware deletes a VMware integration.
func (c *Client) DeleteVMware(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteVMware start")

	_, err := c.Delete(ctx, fmt.Sprintf("/vmware/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting VMware %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteVMware success")
	return nil
}
