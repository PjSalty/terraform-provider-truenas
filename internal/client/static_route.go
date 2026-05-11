package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// StaticRoute represents a static network route in TrueNAS.
type StaticRoute struct {
	ID          int    `json:"id"`
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Description string `json:"description"`
}

// StaticRouteCreateRequest represents the request to create a static route.
type StaticRouteCreateRequest struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Description string `json:"description,omitempty"`
}

// StaticRouteUpdateRequest represents the request to update a static route.
type StaticRouteUpdateRequest struct {
	Destination string `json:"destination,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	Description string `json:"description,omitempty"`
}

// GetStaticRoute retrieves a static route by ID.
func (c *Client) GetStaticRoute(ctx context.Context, id int) (*StaticRoute, error) {
	tflog.Trace(ctx, "GetStaticRoute start")

	resp, err := c.Get(ctx, fmt.Sprintf("/staticroute/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting static route %d: %w", id, err)
	}

	var route StaticRoute
	if err := json.Unmarshal(resp, &route); err != nil {
		return nil, fmt.Errorf("parsing static route response: %w", err)
	}

	tflog.Trace(ctx, "GetStaticRoute success")
	return &route, nil
}

// CreateStaticRoute creates a new static route.
func (c *Client) CreateStaticRoute(ctx context.Context, req *StaticRouteCreateRequest) (*StaticRoute, error) {
	tflog.Trace(ctx, "CreateStaticRoute start")

	resp, err := c.Post(ctx, "/staticroute", req)
	if err != nil {
		return nil, fmt.Errorf("creating static route: %w", err)
	}

	var route StaticRoute
	if err := json.Unmarshal(resp, &route); err != nil {
		return nil, fmt.Errorf("parsing static route create response: %w", err)
	}

	tflog.Trace(ctx, "CreateStaticRoute success")
	return &route, nil
}

// UpdateStaticRoute updates an existing static route.
func (c *Client) UpdateStaticRoute(ctx context.Context, id int, req *StaticRouteUpdateRequest) (*StaticRoute, error) {
	tflog.Trace(ctx, "UpdateStaticRoute start")

	resp, err := c.Put(ctx, fmt.Sprintf("/staticroute/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating static route %d: %w", id, err)
	}

	var route StaticRoute
	if err := json.Unmarshal(resp, &route); err != nil {
		return nil, fmt.Errorf("parsing static route update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateStaticRoute success")
	return &route, nil
}

// DeleteStaticRoute deletes a static route.
func (c *Client) DeleteStaticRoute(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteStaticRoute start")

	_, err := c.Delete(ctx, fmt.Sprintf("/staticroute/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting static route %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteStaticRoute success")
	return nil
}
