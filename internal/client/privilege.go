package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// Privilege, PrivilegeGroup, PrivilegeCreateRequest,
// PrivilegeUpdateRequest moved to internal/types/privilege.go in the
// v2.0 transport-migration prep.
type (
	Privilege              = types.Privilege
	PrivilegeGroup         = types.PrivilegeGroup
	PrivilegeCreateRequest = types.PrivilegeCreateRequest
	PrivilegeUpdateRequest = types.PrivilegeUpdateRequest
)

// ListPrivileges retrieves all privileges.
func (c *Client) ListPrivileges(ctx context.Context) ([]types.Privilege, error) {
	tflog.Trace(ctx, "ListPrivileges start")

	resp, err := c.Get(ctx, "/privilege")
	if err != nil {
		return nil, fmt.Errorf("listing privileges: %w", err)
	}

	var items []types.Privilege
	if err := json.Unmarshal(resp, &items); err != nil {
		return nil, fmt.Errorf("parsing privileges list response: %w", err)
	}
	tflog.Trace(ctx, "ListPrivileges success")
	return items, nil
}

// GetPrivilege retrieves a privilege by ID.
func (c *Client) GetPrivilege(ctx context.Context, id int) (*types.Privilege, error) {
	tflog.Trace(ctx, "GetPrivilege start")

	resp, err := c.Get(ctx, fmt.Sprintf("/privilege/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting privilege %d: %w", id, err)
	}

	var p types.Privilege
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege response: %w", err)
	}
	tflog.Trace(ctx, "GetPrivilege success")
	return &p, nil
}

// CreatePrivilege creates a new privilege.
func (c *Client) CreatePrivilege(ctx context.Context, req *types.PrivilegeCreateRequest) (*types.Privilege, error) {
	tflog.Trace(ctx, "CreatePrivilege start")

	resp, err := c.Post(ctx, "/privilege", req)
	if err != nil {
		return nil, fmt.Errorf("creating privilege: %w", err)
	}

	var p types.Privilege
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege create response: %w", err)
	}
	tflog.Trace(ctx, "CreatePrivilege success")
	return &p, nil
}

// UpdatePrivilege updates an existing privilege.
func (c *Client) UpdatePrivilege(ctx context.Context, id int, req *types.PrivilegeUpdateRequest) (*types.Privilege, error) {
	tflog.Trace(ctx, "UpdatePrivilege start")

	resp, err := c.Put(ctx, fmt.Sprintf("/privilege/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating privilege %d: %w", id, err)
	}

	var p types.Privilege
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege update response: %w", err)
	}
	tflog.Trace(ctx, "UpdatePrivilege success")
	return &p, nil
}

// DeletePrivilege deletes a privilege.
func (c *Client) DeletePrivilege(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeletePrivilege start")

	_, err := c.Delete(ctx, fmt.Sprintf("/privilege/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting privilege %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeletePrivilege success")
	return nil
}
