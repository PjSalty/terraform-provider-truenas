package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for alert services: alertservice.{...}

// GetAlertService retrieves an alert service by ID.
func (c *Client) GetAlertService(ctx context.Context, id int) (*types.AlertService, error) {
	tflog.Trace(ctx, "GetAlertService (ws) start")

	result, err := c.Call(ctx, "alertservice.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting alert service %d: %w", id, err)
	}

	var svc types.AlertService
	if err := json.Unmarshal(result, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service response: %w", err)
	}

	tflog.Trace(ctx, "GetAlertService (ws) success")
	return &svc, nil
}

// CreateAlertService creates a new alert service.
func (c *Client) CreateAlertService(ctx context.Context, req *types.AlertServiceCreateRequest) (*types.AlertService, error) {
	tflog.Trace(ctx, "CreateAlertService (ws) start")

	result, err := c.Call(ctx, "alertservice.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating alert service: %w", err)
	}

	var svc types.AlertService
	if err := json.Unmarshal(result, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service create response: %w", err)
	}

	tflog.Trace(ctx, "CreateAlertService (ws) success")
	return &svc, nil
}

// UpdateAlertService updates an existing alert service.
func (c *Client) UpdateAlertService(ctx context.Context, id int, req *types.AlertServiceUpdateRequest) (*types.AlertService, error) {
	tflog.Trace(ctx, "UpdateAlertService (ws) start")

	result, err := c.Call(ctx, "alertservice.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating alert service %d: %w", id, err)
	}

	var svc types.AlertService
	if err := json.Unmarshal(result, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateAlertService (ws) success")
	return &svc, nil
}

// DeleteAlertService deletes an alert service by ID.
func (c *Client) DeleteAlertService(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteAlertService (ws) start")

	if _, err := c.Call(ctx, "alertservice.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting alert service %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteAlertService (ws) success")
	return nil
}
