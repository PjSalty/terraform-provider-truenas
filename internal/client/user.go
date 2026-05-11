package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ListUsers retrieves all users.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	tflog.Trace(ctx, "ListUsers start")

	resp, err := c.Get(ctx, "/user")
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	var users []User
	if err := json.Unmarshal(resp, &users); err != nil {
		return nil, fmt.Errorf("parsing users list: %w", err)
	}

	tflog.Trace(ctx, "ListUsers success")
	return users, nil
}

// GetUserByUsername finds a user by username.
func (c *Client) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	tflog.Trace(ctx, "GetUserByUsername start")

	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Username == username {
			return &u, nil
		}
	}

	tflog.Trace(ctx, "GetUserByUsername success")
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("user %q not found", username),
	}
}
