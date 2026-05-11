package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Tunable API ---

// Tunable represents a kernel tunable in TrueNAS.
type Tunable struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Var     string `json:"var"`
	Value   string `json:"value"`
	Comment string `json:"comment"`
	Enabled bool   `json:"enabled"`
}

// TunableCreateRequest represents the request to create a tunable.
type TunableCreateRequest struct {
	Type    string `json:"type"`
	Var     string `json:"var"`
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
	Enabled bool   `json:"enabled"`
}

// TunableUpdateRequest represents the request to update a tunable.
type TunableUpdateRequest struct {
	Value   string `json:"value,omitempty"`
	Comment string `json:"comment,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// GetTunable retrieves a tunable by ID.
func (c *Client) GetTunable(ctx context.Context, id int) (*Tunable, error) {
	tflog.Trace(ctx, "GetTunable start")

	resp, err := c.Get(ctx, fmt.Sprintf("/tunable/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting tunable %d: %w", id, err)
	}

	var tunable Tunable
	if err := json.Unmarshal(resp, &tunable); err != nil {
		return nil, fmt.Errorf("parsing tunable response: %w", err)
	}

	tflog.Trace(ctx, "GetTunable success")
	return &tunable, nil
}

// CreateTunable creates a new tunable.
func (c *Client) CreateTunable(ctx context.Context, req *TunableCreateRequest) (*Tunable, error) {
	tflog.Trace(ctx, "CreateTunable start")

	_, err := c.Post(ctx, "/tunable", req)
	if err != nil {
		return nil, fmt.Errorf("creating tunable: %w", err)
	}

	// The create endpoint returns an internal ID that does not match the tunable ID.
	// Find the tunable by its var name instead.
	tflog.Trace(ctx, "CreateTunable success")
	return c.FindTunableByVar(ctx, req.Var)
}

// FindTunableByVar finds a tunable by its variable name.
func (c *Client) FindTunableByVar(ctx context.Context, varName string) (*Tunable, error) {
	tflog.Trace(ctx, "FindTunableByVar start")

	tunables, err := c.ListTunables(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tunables {
		if tunables[i].Var == varName {
			return &tunables[i], nil
		}
	}
	tflog.Trace(ctx, "FindTunableByVar success")
	return nil, fmt.Errorf("tunable with var %q not found after creation", varName)
}

// ListTunables retrieves all tunables.
func (c *Client) ListTunables(ctx context.Context) ([]Tunable, error) {
	tflog.Trace(ctx, "ListTunables start")

	resp, err := c.Get(ctx, "/tunable")
	if err != nil {
		return nil, fmt.Errorf("listing tunables: %w", err)
	}

	var tunables []Tunable
	if err := json.Unmarshal(resp, &tunables); err != nil {
		return nil, fmt.Errorf("parsing tunables list: %w", err)
	}

	tflog.Trace(ctx, "ListTunables success")
	return tunables, nil
}

// UpdateTunable updates an existing tunable.
func (c *Client) UpdateTunable(ctx context.Context, id int, req *TunableUpdateRequest) (*Tunable, error) {
	tflog.Trace(ctx, "UpdateTunable start")

	_, err := c.Put(ctx, fmt.Sprintf("/tunable/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating tunable %d: %w", id, err)
	}

	// The update endpoint returns an internal ID, not the tunable ID.
	// Re-fetch by the original ID which remains stable.
	tflog.Trace(ctx, "UpdateTunable success")
	return c.GetTunable(ctx, id)
}

// DeleteTunable deletes a tunable.
func (c *Client) DeleteTunable(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteTunable start")

	_, err := c.Delete(ctx, fmt.Sprintf("/tunable/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting tunable %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteTunable success")
	return nil
}
