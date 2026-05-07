package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ListGroups retrieves all groups.
func (c *Client) ListGroups(ctx context.Context) ([]types.Group, error) {
	tflog.Trace(ctx, "ListGroups start")

	resp, err := c.Get(ctx, "/group")
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
	}

	var groups []types.Group
	if err := json.Unmarshal(resp, &groups); err != nil {
		return nil, fmt.Errorf("parsing groups list: %w", err)
	}

	tflog.Trace(ctx, "ListGroups success")
	return groups, nil
}

// GetGroupByName finds a group by name.
func (c *Client) GetGroupByName(ctx context.Context, name string) (*types.Group, error) {
	tflog.Trace(ctx, "GetGroupByName start")

	groups, err := c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Name == name {
			return &g, nil
		}
	}

	tflog.Trace(ctx, "GetGroupByName success")
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("group %q not found", name),
	}
}
