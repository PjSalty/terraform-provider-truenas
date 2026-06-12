package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for groups: group.{...}.

// ListGroups retrieves all groups.
func (c *Client) ListGroups(ctx context.Context) ([]types.Group, error) {
	tflog.Trace(ctx, "ListGroups (ws) start")

	result, err := c.Call(ctx, "group.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
	}

	var groups []types.Group
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("parsing groups list: %w", err)
	}

	tflog.Trace(ctx, "ListGroups (ws) success")
	return groups, nil
}

// GetGroup retrieves a group by ID.
func (c *Client) GetGroup(ctx context.Context, id int) (*types.Group, error) {
	tflog.Trace(ctx, "GetGroup (ws) start")

	result, err := c.Call(ctx, "group.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting group %d: %w", id, err)
	}

	var group types.Group
	if err := json.Unmarshal(result, &group); err != nil {
		return nil, fmt.Errorf("parsing group response: %w", err)
	}

	tflog.Trace(ctx, "GetGroup (ws) success")
	return &group, nil
}

// GetGroupByName finds a group by name.
func (c *Client) GetGroupByName(ctx context.Context, name string) (*types.Group, error) {
	tflog.Trace(ctx, "GetGroupByName (ws) start")

	groups, err := c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Name == name {
			return &g, nil
		}
	}

	tflog.Trace(ctx, "GetGroupByName (ws) success")
	return nil, &RPCError{
		Code:    CodeInvalidParams,
		Message: fmt.Sprintf("group %q not found", name),
	}
}

// CreateGroup creates a new local group. The TrueNAS group.create method
// returns the full group object directly (unlike REST, which returns just
// the ID and requires a follow-up GET).
func (c *Client) CreateGroup(ctx context.Context, req *types.GroupCreateRequest) (*types.Group, error) {
	tflog.Trace(ctx, "CreateGroup (ws) start")

	result, err := c.Call(ctx, "group.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating group %q: %w", req.Name, err)
	}

	// Some TrueNAS versions return just the group ID; others return the
	// full object. Try the object first, fall back to ID + re-fetch.
	var group types.Group
	if err := json.Unmarshal(result, &group); err == nil && group.ID != 0 {
		tflog.Trace(ctx, "CreateGroup (ws) success")
		return &group, nil
	}

	var groupID int
	if err := json.Unmarshal(result, &groupID); err != nil {
		return nil, fmt.Errorf("parsing group create response: %w", err)
	}

	tflog.Trace(ctx, "CreateGroup (ws) success")
	return c.GetGroup(ctx, groupID)
}

// UpdateGroup updates an existing group.
func (c *Client) UpdateGroup(ctx context.Context, id int, req *types.GroupUpdateRequest) (*types.Group, error) {
	tflog.Trace(ctx, "UpdateGroup (ws) start")

	result, err := c.Call(ctx, "group.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating group %d: %w", id, err)
	}

	var group types.Group
	if err := json.Unmarshal(result, &group); err == nil && group.ID != 0 {
		tflog.Trace(ctx, "UpdateGroup (ws) success")
		return &group, nil
	}

	var groupID int
	if err := json.Unmarshal(result, &groupID); err != nil {
		return nil, fmt.Errorf("parsing group update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateGroup (ws) success")
	return c.GetGroup(ctx, groupID)
}

// DeleteGroup deletes a group. The TrueNAS group.delete method takes
// (id, options) where options.delete_users=false matches the REST
// behavior.
func (c *Client) DeleteGroup(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteGroup (ws) start")

	if _, err := c.Call(ctx, "group.delete",
		[]interface{}{id, map[string]bool{"delete_users": false}},
		CallOptions{}); err != nil {
		return fmt.Errorf("deleting group %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteGroup (ws) success")
	return nil
}
