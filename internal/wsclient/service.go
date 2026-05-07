package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for system services:
// service.{query, get_instance, update, start, stop}.

// ListServices retrieves all services.
func (c *Client) ListServices(ctx context.Context) ([]types.Service, error) {
	tflog.Trace(ctx, "ListServices (ws) start")

	result, err := c.Call(ctx, "service.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	var services []types.Service
	if err := json.Unmarshal(result, &services); err != nil {
		return nil, fmt.Errorf("parsing services list: %w", err)
	}

	tflog.Trace(ctx, "ListServices (ws) success")
	return services, nil
}

// GetService retrieves a service by ID.
func (c *Client) GetService(ctx context.Context, id int) (*types.Service, error) {
	tflog.Trace(ctx, "GetService (ws) start")

	result, err := c.Call(ctx, "service.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting service %d: %w", id, err)
	}

	var svc types.Service
	if err := json.Unmarshal(result, &svc); err != nil {
		return nil, fmt.Errorf("parsing service response: %w", err)
	}

	tflog.Trace(ctx, "GetService (ws) success")
	return &svc, nil
}

// GetServiceByName uses server-side filtering on service.query for an
// O(1) lookup.
func (c *Client) GetServiceByName(ctx context.Context, name string) (*types.Service, error) {
	tflog.Trace(ctx, "GetServiceByName (ws) start")

	filters := []interface{}{[]interface{}{"service", "=", name}}
	result, err := c.Call(ctx, "service.query",
		[]interface{}{filters},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	var services []types.Service
	if err := json.Unmarshal(result, &services); err != nil {
		return nil, fmt.Errorf("parsing services list: %w", err)
	}

	if len(services) == 0 {
		return nil, &RPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("service %q not found", name),
		}
	}

	tflog.Trace(ctx, "GetServiceByName (ws) success")
	return &services[0], nil
}

// UpdateService updates a service's enable flag.
func (c *Client) UpdateService(ctx context.Context, id int, req *types.ServiceUpdateRequest) error {
	tflog.Trace(ctx, "UpdateService (ws) start")

	if _, err := c.Call(ctx, "service.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("updating service %d: %w", id, err)
	}

	tflog.Trace(ctx, "UpdateService (ws) success")
	return nil
}

// StartService starts a service by name.
func (c *Client) StartService(ctx context.Context, name string) error {
	tflog.Trace(ctx, "StartService (ws) start")

	if _, err := c.Call(ctx, "service.start",
		[]interface{}{name},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("starting service %q: %w", name, err)
	}

	tflog.Trace(ctx, "StartService (ws) success")
	return nil
}

// StopService stops a service by name.
func (c *Client) StopService(ctx context.Context, name string) error {
	tflog.Trace(ctx, "StopService (ws) start")

	if _, err := c.Call(ctx, "service.stop",
		[]interface{}{name},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("stopping service %q: %w", name, err)
	}

	tflog.Trace(ctx, "StopService (ws) success")
	return nil
}
