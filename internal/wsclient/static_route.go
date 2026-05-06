package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for static routes: staticroute.{...}

// GetStaticRoute retrieves a static route by ID.
func (c *Client) GetStaticRoute(ctx context.Context, id int) (*types.StaticRoute, error) {
	tflog.Trace(ctx, "GetStaticRoute (ws) start")

	result, err := c.Call(ctx, "staticroute.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting static route %d: %w", id, err)
	}

	var route types.StaticRoute
	if err := json.Unmarshal(result, &route); err != nil {
		return nil, fmt.Errorf("parsing static route response: %w", err)
	}

	tflog.Trace(ctx, "GetStaticRoute (ws) success")
	return &route, nil
}

// CreateStaticRoute creates a new static route.
func (c *Client) CreateStaticRoute(ctx context.Context, req *types.StaticRouteCreateRequest) (*types.StaticRoute, error) {
	tflog.Trace(ctx, "CreateStaticRoute (ws) start")

	result, err := c.Call(ctx, "staticroute.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating static route: %w", err)
	}

	var route types.StaticRoute
	if err := json.Unmarshal(result, &route); err != nil {
		return nil, fmt.Errorf("parsing static route create response: %w", err)
	}

	tflog.Trace(ctx, "CreateStaticRoute (ws) success")
	return &route, nil
}

// UpdateStaticRoute updates an existing static route.
func (c *Client) UpdateStaticRoute(ctx context.Context, id int, req *types.StaticRouteUpdateRequest) (*types.StaticRoute, error) {
	tflog.Trace(ctx, "UpdateStaticRoute (ws) start")

	result, err := c.Call(ctx, "staticroute.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating static route %d: %w", id, err)
	}

	var route types.StaticRoute
	if err := json.Unmarshal(result, &route); err != nil {
		return nil, fmt.Errorf("parsing static route update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateStaticRoute (ws) success")
	return &route, nil
}

// DeleteStaticRoute deletes a static route by ID.
func (c *Client) DeleteStaticRoute(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteStaticRoute (ws) start")

	if _, err := c.Call(ctx, "staticroute.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting static route %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteStaticRoute (ws) success")
	return nil
}
