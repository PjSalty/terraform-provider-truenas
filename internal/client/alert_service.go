package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- Alert Service API ---

// AlertService, AlertServiceCreateRequest, AlertServiceUpdateRequest moved
// to internal/types/alert_service.go in the v2.0 transport-migration prep.
type (
	AlertService              = types.AlertService
	AlertServiceCreateRequest = types.AlertServiceCreateRequest
	AlertServiceUpdateRequest = types.AlertServiceUpdateRequest
)

// GetAlertService retrieves an alert service by ID.
func (c *Client) GetAlertService(ctx context.Context, id int) (*types.AlertService, error) {
	tflog.Trace(ctx, "GetAlertService start")

	resp, err := c.Get(ctx, fmt.Sprintf("/alertservice/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting alert service %d: %w", id, err)
	}

	var svc types.AlertService
	if err := json.Unmarshal(resp, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service response: %w", err)
	}

	tflog.Trace(ctx, "GetAlertService success")
	return &svc, nil
}

// CreateAlertService creates a new alert service.
func (c *Client) CreateAlertService(ctx context.Context, req *types.AlertServiceCreateRequest) (*types.AlertService, error) {
	tflog.Trace(ctx, "CreateAlertService start")

	resp, err := c.Post(ctx, "/alertservice", req)
	if err != nil {
		return nil, fmt.Errorf("creating alert service: %w", err)
	}

	var svc types.AlertService
	if err := json.Unmarshal(resp, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service create response: %w", err)
	}

	tflog.Trace(ctx, "CreateAlertService success")
	return &svc, nil
}

// UpdateAlertService updates an existing alert service.
func (c *Client) UpdateAlertService(ctx context.Context, id int, req *types.AlertServiceUpdateRequest) (*types.AlertService, error) {
	tflog.Trace(ctx, "UpdateAlertService start")

	resp, err := c.Put(ctx, fmt.Sprintf("/alertservice/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating alert service %d: %w", id, err)
	}

	var svc types.AlertService
	if err := json.Unmarshal(resp, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateAlertService success")
	return &svc, nil
}

// DeleteAlertService deletes an alert service.
func (c *Client) DeleteAlertService(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteAlertService start")

	_, err := c.Delete(ctx, fmt.Sprintf("/alertservice/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting alert service %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteAlertService success")
	return nil
}
