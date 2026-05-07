package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for ZFS pools:
// pool.{query, get_instance, create, export}.
//
// pool.create and pool.export are jobs server-side (vdev assembly +
// resilver scheduling, or zpool export + filesystem teardown). CallJob
// waits for terminal state. After job completion, this client refetches
// via pool.query (filtered by name) because the job result may not
// contain the placed pool record on every SCALE point release.
const poolPollInterval = 1 * time.Second

// GetPool retrieves a pool by ID.
func (c *Client) GetPool(ctx context.Context, id int) (*types.Pool, error) {
	tflog.Trace(ctx, "GetPool (ws) start")

	result, err := c.Call(ctx, "pool.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting pool %d: %w", id, err)
	}

	var pool types.Pool
	if err := json.Unmarshal(result, &pool); err != nil {
		return nil, fmt.Errorf("parsing pool response: %w", err)
	}

	tflog.Trace(ctx, "GetPool (ws) success")
	return &pool, nil
}

// ListPools retrieves all pools.
func (c *Client) ListPools(ctx context.Context) ([]types.Pool, error) {
	tflog.Trace(ctx, "ListPools (ws) start")

	result, err := c.Call(ctx, "pool.query", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing pools: %w", err)
	}

	var pools []types.Pool
	if err := json.Unmarshal(result, &pools); err != nil {
		return nil, fmt.Errorf("parsing pools list: %w", err)
	}

	tflog.Trace(ctx, "ListPools (ws) success")
	return pools, nil
}

// GetPoolByName uses server-side filtering on pool.query for an O(1)
// lookup, instead of the REST client's list-then-filter.
func (c *Client) GetPoolByName(ctx context.Context, name string) (*types.Pool, error) {
	tflog.Trace(ctx, "GetPoolByName (ws) start")

	filters := []interface{}{[]interface{}{"name", "=", name}}
	result, err := c.Call(ctx, "pool.query",
		[]interface{}{filters},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing pools: %w", err)
	}

	var pools []types.Pool
	if err := json.Unmarshal(result, &pools); err != nil {
		return nil, fmt.Errorf("parsing pools list: %w", err)
	}

	if len(pools) == 0 {
		return nil, &RPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("pool %q not found", name),
		}
	}

	tflog.Trace(ctx, "GetPoolByName (ws) success")
	return &pools[0], nil
}

// CreatePool creates a new ZFS pool. The underlying pool.create RPC is
// a job (vdev assembly takes seconds-to-minutes for large pools).
// CallJob waits for terminal state, then this client looks up the
// placed pool by name — relying on the job's result field is unsafe
// across SCALE point releases.
func (c *Client) CreatePool(ctx context.Context, req *types.PoolCreateRequest) (*types.Pool, error) {
	tflog.Trace(ctx, "CreatePool (ws) start")

	jobResult, err := c.CallJob(ctx, "pool.create",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: false},
		poolPollInterval)
	if err != nil {
		return nil, fmt.Errorf("creating pool %q: %w", req.Name, err)
	}

	// Best-effort: if the job result contained the placed pool, return
	// it directly to avoid a follow-on round-trip.
	if len(jobResult) > 0 && string(jobResult) != "null" {
		var pool types.Pool
		if err := json.Unmarshal(jobResult, &pool); err == nil && pool.ID != 0 {
			tflog.Trace(ctx, "CreatePool (ws) success (from job result)")
			return &pool, nil
		}
	}

	// Fallback: server didn't return the pool in the job result;
	// look it up by name.
	tflog.Trace(ctx, "CreatePool (ws) success (refetch)")
	return c.GetPoolByName(ctx, req.Name)
}

// ExportPool exports (and optionally destroys) a ZFS pool. This is the
// resource's delete operation. Job-bound on the server.
func (c *Client) ExportPool(ctx context.Context, id int, req *types.PoolExportRequest) error {
	tflog.Trace(ctx, "ExportPool (ws) start")

	_, err := c.CallJob(ctx, "pool.export",
		[]interface{}{id, req},
		CallOptions{Job: true, Idempotent: false},
		poolPollInterval)
	if err != nil {
		return fmt.Errorf("exporting pool %d: %w", id, err)
	}

	tflog.Trace(ctx, "ExportPool (ws) success")
	return nil
}
