package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method names for kernel tunables. The TrueNAS WebSocket API
// follows the convention REST-path -> dot-syntax, so /tunable becomes
// "tunable.*". Same shape applies to the params arrays.

// GetTunable retrieves a tunable by ID.
func (c *Client) GetTunable(ctx context.Context, id int) (*types.Tunable, error) {
	tflog.Trace(ctx, "GetTunable (ws) start")

	result, err := c.Call(ctx, "tunable.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting tunable %d: %w", id, err)
	}

	var tun types.Tunable
	if err := json.Unmarshal(result, &tun); err != nil {
		return nil, fmt.Errorf("parsing tunable response: %w", err)
	}

	tflog.Trace(ctx, "GetTunable (ws) success")
	return &tun, nil
}

// tunablePollInterval is the cadence at which we poll core.get_jobs
// for tunable.{create,update} completion. Tunables apply nearly
// instantly server-side; 500ms is plenty of headroom without making
// the operator wait an extra second per resource.
const tunablePollInterval = 500 * time.Millisecond

// CreateTunable creates a new tunable. SCALE 25.10 made tunable.create
// a long-running job (the synchronous return from the REST era is
// gone). CallJob polls core.get_jobs until terminal state and returns
// the created Tunable from the job's result field.
func (c *Client) CreateTunable(ctx context.Context, req *types.TunableCreateRequest) (*types.Tunable, error) {
	tflog.Trace(ctx, "CreateTunable (ws) start")

	jobResult, err := c.CallJob(ctx, "tunable.create",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: false},
		tunablePollInterval)
	if err != nil {
		return nil, fmt.Errorf("creating tunable: %w", err)
	}

	var tun types.Tunable
	if err := json.Unmarshal(jobResult, &tun); err != nil {
		return nil, fmt.Errorf("parsing tunable from job result: %w", err)
	}

	tflog.Trace(ctx, "CreateTunable (ws) success")
	return &tun, nil
}

// FindTunableByVar finds a tunable by its variable name. Retained for
// API parity with the REST client even though the WebSocket
// CreateTunable does not need it; resource Read paths that key off
// var-name still rely on this.
func (c *Client) FindTunableByVar(ctx context.Context, varName string) (*types.Tunable, error) {
	tflog.Trace(ctx, "FindTunableByVar (ws) start")

	tunables, err := c.ListTunables(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tunables {
		if tunables[i].Var == varName {
			return &tunables[i], nil
		}
	}
	tflog.Trace(ctx, "FindTunableByVar (ws) not-found")
	return nil, fmt.Errorf("tunable with var %q not found after creation", varName)
}

// ListTunables retrieves all tunables via tunable.query with no filters.
// JSON-RPC query takes [filters, options]; we send empty arrays for both.
func (c *Client) ListTunables(ctx context.Context) ([]types.Tunable, error) {
	tflog.Trace(ctx, "ListTunables (ws) start")

	result, err := c.Call(ctx, "tunable.query",
		[]interface{}{[]interface{}{}, map[string]interface{}{}},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing tunables: %w", err)
	}

	var tunables []types.Tunable
	if err := json.Unmarshal(result, &tunables); err != nil {
		return nil, fmt.Errorf("parsing tunables list: %w", err)
	}

	tflog.Trace(ctx, "ListTunables (ws) success")
	return tunables, nil
}

// UpdateTunable updates an existing tunable. Like create, SCALE 25.10
// surfaced this as a long-running job on the WS path.
func (c *Client) UpdateTunable(ctx context.Context, id int, req *types.TunableUpdateRequest) (*types.Tunable, error) {
	tflog.Trace(ctx, "UpdateTunable (ws) start")

	jobResult, err := c.CallJob(ctx, "tunable.update",
		[]interface{}{id, req},
		CallOptions{Job: true, Idempotent: false},
		tunablePollInterval)
	if err != nil {
		return nil, fmt.Errorf("updating tunable %d: %w", id, err)
	}

	var tun types.Tunable
	if err := json.Unmarshal(jobResult, &tun); err != nil {
		return nil, fmt.Errorf("parsing tunable from job result: %w", err)
	}

	tflog.Trace(ctx, "UpdateTunable (ws) success")
	return &tun, nil
}

// DeleteTunable deletes a tunable by ID. Same async-job pattern as
// the rest of the tunable.* surface on SCALE 25.10.
func (c *Client) DeleteTunable(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteTunable (ws) start")

	if _, err := c.CallJob(ctx, "tunable.delete",
		[]interface{}{id},
		CallOptions{Job: true, Idempotent: true},
		tunablePollInterval); err != nil {
		return fmt.Errorf("deleting tunable %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteTunable (ws) success")
	return nil
}
