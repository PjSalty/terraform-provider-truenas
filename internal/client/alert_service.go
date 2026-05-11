package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Alert Service API ---

// AlertService represents an alert service configuration.
//
// SCALE version notes:
//   - 25.04 (legacy): top-level `type` field in both request and response.
//   - 25.10+ (current): `type` lives inside `attributes` as the polymorphic
//     discriminator (per the `oneOf` schema). The REST endpoint rejects a
//     top-level `type` with "Extra inputs are not permitted".
//
// We model the struct for 25.10+ and require callers to put `type` in
// Settings. GetType() reads it back from either location.
type AlertService struct {
	ID       int                    `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type,omitempty"`
	Enabled  bool                   `json:"enabled"`
	Level    string                 `json:"level"`
	Settings map[string]interface{} `json:"attributes"`
}

// GetType returns the alert service type, preferring the attributes embedded
// form (SCALE 25.10+) and falling back to the 25.04 top-level field.
func (a *AlertService) GetType() string {
	if a.Settings != nil {
		if t, ok := a.Settings["type"].(string); ok && t != "" {
			return t
		}
	}
	if a.Type != "" {
		return a.Type
	}
	return ""
}

// AlertServiceCreateRequest represents the request to create an alert service.
// On SCALE 25.10+ `type` MUST be embedded inside attributes (the discriminator
// field of the `oneOf` schema); top-level `type` is rejected. Callers should
// not set the top-level Type field — it exists only as a fallback for any
// pre-25.04 backends.
type AlertServiceCreateRequest struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Level    string                 `json:"level"`
	Settings map[string]interface{} `json:"attributes"`
}

// AlertServiceUpdateRequest represents the request to update an alert service.
type AlertServiceUpdateRequest struct {
	Name     string                 `json:"name,omitempty"`
	Enabled  *bool                  `json:"enabled,omitempty"`
	Level    string                 `json:"level,omitempty"`
	Settings map[string]interface{} `json:"attributes,omitempty"`
}

// GetAlertService retrieves an alert service by ID.
func (c *Client) GetAlertService(ctx context.Context, id int) (*AlertService, error) {
	tflog.Trace(ctx, "GetAlertService start")

	resp, err := c.Get(ctx, fmt.Sprintf("/alertservice/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting alert service %d: %w", id, err)
	}

	var svc AlertService
	if err := json.Unmarshal(resp, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service response: %w", err)
	}

	tflog.Trace(ctx, "GetAlertService success")
	return &svc, nil
}

// CreateAlertService creates a new alert service.
func (c *Client) CreateAlertService(ctx context.Context, req *AlertServiceCreateRequest) (*AlertService, error) {
	tflog.Trace(ctx, "CreateAlertService start")

	resp, err := c.Post(ctx, "/alertservice", req)
	if err != nil {
		return nil, fmt.Errorf("creating alert service: %w", err)
	}

	var svc AlertService
	if err := json.Unmarshal(resp, &svc); err != nil {
		return nil, fmt.Errorf("parsing alert service create response: %w", err)
	}

	tflog.Trace(ctx, "CreateAlertService success")
	return &svc, nil
}

// UpdateAlertService updates an existing alert service.
func (c *Client) UpdateAlertService(ctx context.Context, id int, req *AlertServiceUpdateRequest) (*AlertService, error) {
	tflog.Trace(ctx, "UpdateAlertService start")

	resp, err := c.Put(ctx, fmt.Sprintf("/alertservice/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating alert service %d: %w", id, err)
	}

	var svc AlertService
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
