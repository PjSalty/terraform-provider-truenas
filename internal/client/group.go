package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ListGroups retrieves all groups.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	tflog.Trace(ctx, "ListGroups start")

	resp, err := c.Get(ctx, "/group")
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
	}

	var groups []Group
	if err := json.Unmarshal(resp, &groups); err != nil {
		return nil, fmt.Errorf("parsing groups list: %w", err)
	}

	tflog.Trace(ctx, "ListGroups success")
	return groups, nil
}

// GetGroupByName finds a group by name.
func (c *Client) GetGroupByName(ctx context.Context, name string) (*Group, error) {
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
