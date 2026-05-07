package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for users: user.{...}.

// ListUsers retrieves all users.
func (c *Client) ListUsers(ctx context.Context) ([]types.User, error) {
	tflog.Trace(ctx, "ListUsers (ws) start")

	result, err := c.Call(ctx, "user.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	var users []types.User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, fmt.Errorf("parsing users list: %w", err)
	}

	tflog.Trace(ctx, "ListUsers (ws) success")
	return users, nil
}

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, id int) (*types.User, error) {
	tflog.Trace(ctx, "GetUser (ws) start")

	result, err := c.Call(ctx, "user.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting user %d: %w", id, err)
	}

	var user types.User
	if err := json.Unmarshal(result, &user); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}

	tflog.Trace(ctx, "GetUser (ws) success")
	return &user, nil
}

// GetUserByUsername finds a user by username.
func (c *Client) GetUserByUsername(ctx context.Context, username string) (*types.User, error) {
	tflog.Trace(ctx, "GetUserByUsername (ws) start")

	users, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Username == username {
			return &u, nil
		}
	}

	tflog.Trace(ctx, "GetUserByUsername (ws) success")
	return nil, &RPCError{
		Code:    CodeInvalidParams,
		Message: fmt.Sprintf("user %q not found", username),
	}
}

// CreateUser creates a new local user.
func (c *Client) CreateUser(ctx context.Context, req *types.UserCreateRequest) (*types.User, error) {
	tflog.Trace(ctx, "CreateUser (ws) start")

	result, err := c.Call(ctx, "user.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating user %q: %w", req.Username, err)
	}

	var user types.User
	if err := json.Unmarshal(result, &user); err != nil {
		return nil, fmt.Errorf("parsing user create response: %w", err)
	}

	tflog.Trace(ctx, "CreateUser (ws) success")
	return &user, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, id int, req *types.UserUpdateRequest) (*types.User, error) {
	tflog.Trace(ctx, "UpdateUser (ws) start")

	result, err := c.Call(ctx, "user.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating user %d: %w", id, err)
	}

	var user types.User
	if err := json.Unmarshal(result, &user); err != nil {
		return nil, fmt.Errorf("parsing user update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateUser (ws) success")
	return &user, nil
}

// DeleteUser deletes a user. The TrueNAS user.delete method takes
// (id, options) where options.delete_group=true matches the REST
// behavior.
func (c *Client) DeleteUser(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteUser (ws) start")

	if _, err := c.Call(ctx, "user.delete",
		[]interface{}{id, map[string]bool{"delete_group": true}},
		CallOptions{}); err != nil {
		return fmt.Errorf("deleting user %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteUser (ws) success")
	return nil
}
