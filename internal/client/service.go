package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Service represents a TrueNAS service.
type Service struct {
	ID      int    `json:"id"`
	Service string `json:"service"`
	Enable  bool   `json:"enable"`
	State   string `json:"state"`
	Pids    []int  `json:"pids"`
}

// ServiceUpdateRequest represents the request to update a service.
type ServiceUpdateRequest struct {
	Enable bool `json:"enable"`
}

// ServiceStartStopRequest represents the request to start or stop a service.
type ServiceStartStopRequest struct {
	Service string `json:"service"`
}

// ListServices retrieves all services.
func (c *Client) ListServices(ctx context.Context) ([]Service, error) {
	tflog.Trace(ctx, "ListServices start")

	resp, err := c.Get(ctx, "/service")
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	var services []Service
	if err := json.Unmarshal(resp, &services); err != nil {
		return nil, fmt.Errorf("parsing services list: %w", err)
	}

	tflog.Trace(ctx, "ListServices success")
	return services, nil
}

// GetService retrieves a service by ID.
func (c *Client) GetService(ctx context.Context, id int) (*Service, error) {
	tflog.Trace(ctx, "GetService start")

	resp, err := c.Get(ctx, fmt.Sprintf("/service/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting service %d: %w", id, err)
	}

	var svc Service
	if err := json.Unmarshal(resp, &svc); err != nil {
		return nil, fmt.Errorf("parsing service response: %w", err)
	}

	tflog.Trace(ctx, "GetService success")
	return &svc, nil
}

// GetServiceByName finds a service by its name.
func (c *Client) GetServiceByName(ctx context.Context, name string) (*Service, error) {
	tflog.Trace(ctx, "GetServiceByName start")

	services, err := c.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	for _, svc := range services {
		if svc.Service == name {
			return &svc, nil
		}
	}

	tflog.Trace(ctx, "GetServiceByName success")
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("service %q not found", name),
	}
}

// UpdateService updates a service's enable flag.
func (c *Client) UpdateService(ctx context.Context, id int, req *ServiceUpdateRequest) error {
	tflog.Trace(ctx, "UpdateService start")

	_, err := c.Put(ctx, fmt.Sprintf("/service/id/%d", id), req)
	if err != nil {
		return fmt.Errorf("updating service %d: %w", id, err)
	}
	tflog.Trace(ctx, "UpdateService success")
	return nil
}

// StartService starts a service by name.
func (c *Client) StartService(ctx context.Context, name string) error {
	tflog.Trace(ctx, "StartService start")

	_, err := c.Post(ctx, "/service/start", &ServiceStartStopRequest{Service: name})
	if err != nil {
		return fmt.Errorf("starting service %q: %w", name, err)
	}
	tflog.Trace(ctx, "StartService success")
	return nil
}

// StopService stops a service by name.
func (c *Client) StopService(ctx context.Context, name string) error {
	tflog.Trace(ctx, "StopService start")

	_, err := c.Post(ctx, "/service/stop", &ServiceStartStopRequest{Service: name})
	if err != nil {
		return fmt.Errorf("stopping service %q: %w", name, err)
	}
	tflog.Trace(ctx, "StopService success")
	return nil
}
