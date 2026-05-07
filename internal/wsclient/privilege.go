package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for privileges: privilege.{...}.

// ListPrivileges retrieves all privileges.
func (c *Client) ListPrivileges(ctx context.Context) ([]types.Privilege, error) {
	tflog.Trace(ctx, "ListPrivileges (ws) start")

	result, err := c.Call(ctx, "privilege.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing privileges: %w", err)
	}

	var items []types.Privilege
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("parsing privileges list response: %w", err)
	}

	tflog.Trace(ctx, "ListPrivileges (ws) success")
	return items, nil
}

// GetPrivilege retrieves a privilege by ID.
func (c *Client) GetPrivilege(ctx context.Context, id int) (*types.Privilege, error) {
	tflog.Trace(ctx, "GetPrivilege (ws) start")

	result, err := c.Call(ctx, "privilege.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting privilege %d: %w", id, err)
	}

	var p types.Privilege
	if err := json.Unmarshal(result, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege response: %w", err)
	}

	tflog.Trace(ctx, "GetPrivilege (ws) success")
	return &p, nil
}

// CreatePrivilege creates a new privilege.
func (c *Client) CreatePrivilege(ctx context.Context, req *types.PrivilegeCreateRequest) (*types.Privilege, error) {
	tflog.Trace(ctx, "CreatePrivilege (ws) start")

	result, err := c.Call(ctx, "privilege.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating privilege: %w", err)
	}

	var p types.Privilege
	if err := json.Unmarshal(result, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege create response: %w", err)
	}

	tflog.Trace(ctx, "CreatePrivilege (ws) success")
	return &p, nil
}

// UpdatePrivilege updates an existing privilege.
func (c *Client) UpdatePrivilege(ctx context.Context, id int, req *types.PrivilegeUpdateRequest) (*types.Privilege, error) {
	tflog.Trace(ctx, "UpdatePrivilege (ws) start")

	result, err := c.Call(ctx, "privilege.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating privilege %d: %w", id, err)
	}

	var p types.Privilege
	if err := json.Unmarshal(result, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege update response: %w", err)
	}

	tflog.Trace(ctx, "UpdatePrivilege (ws) success")
	return &p, nil
}

// DeletePrivilege deletes a privilege.
func (c *Client) DeletePrivilege(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeletePrivilege (ws) start")

	if _, err := c.Call(ctx, "privilege.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting privilege %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeletePrivilege (ws) success")
	return nil
}
