package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- Pool API ---
//
// Read-only Pool/GetPool/ListPools live at the top of this file. The
// create/update/export (delete) operations below are asynchronous and use
// the job polling mechanism via WaitForJob.

// Pool, PoolCreateRequest, PoolExportRequest moved to
// internal/types/pool.go in the v2.0 transport-migration prep.
type Pool = types.Pool

// GetPool retrieves a pool by ID.
func (c *Client) GetPool(ctx context.Context, id int) (*Pool, error) {
	tflog.Trace(ctx, "GetPool start")

	resp, err := c.Get(ctx, fmt.Sprintf("/pool/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting pool %d: %w", id, err)
	}

	var pool Pool
	if err := json.Unmarshal(resp, &pool); err != nil {
		return nil, fmt.Errorf("parsing pool response: %w", err)
	}

	tflog.Trace(ctx, "GetPool success")
	return &pool, nil
}

// ListPools retrieves all pools.
func (c *Client) ListPools(ctx context.Context) ([]Pool, error) {
	tflog.Trace(ctx, "ListPools start")

	resp, err := c.Get(ctx, "/pool")
	if err != nil {
		return nil, fmt.Errorf("listing pools: %w", err)
	}

	var pools []Pool
	if err := json.Unmarshal(resp, &pools); err != nil {
		return nil, fmt.Errorf("parsing pools list: %w", err)
	}

	tflog.Trace(ctx, "ListPools success")
	return pools, nil
}

type (
	PoolCreateRequest = types.PoolCreateRequest
	PoolExportRequest = types.PoolExportRequest
)

// CreatePool creates a new ZFS pool. This is an asynchronous operation
// that returns a job ID; we wait for the job to complete and unmarshal
// the resulting pool from the job's result field.
func (c *Client) CreatePool(ctx context.Context, req *PoolCreateRequest) (*Pool, error) {
	tflog.Trace(ctx, "CreatePool start")

	resp, err := c.Post(ctx, "/pool", req)
	if err != nil {
		return nil, fmt.Errorf("creating pool %q: %w", req.Name, err)
	}

	var jobID int
	if err := json.Unmarshal(resp, &jobID); err != nil {
		// Maybe the API returned the pool directly
		var pool Pool
		if err2 := json.Unmarshal(resp, &pool); err2 == nil {
			return &pool, nil
		}
		return nil, fmt.Errorf("parsing pool create job id: %w", err)
	}

	job, err := c.WaitForJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("waiting for pool creation: %w", err)
	}

	var pool Pool
	if len(job.Result) > 0 && string(job.Result) != "null" {
		if err := json.Unmarshal(job.Result, &pool); err != nil {
			return nil, fmt.Errorf("parsing pool from job result: %w", err)
		}
	}

	// If the job result didn't contain the pool, look it up by name.
	if pool.ID == 0 {
		pools, err := c.ListPools(ctx)
		if err != nil {
			return nil, fmt.Errorf("looking up created pool: %w", err)
		}
		for _, p := range pools {
			if p.Name == req.Name {
				pool = p
				break
			}
		}
	}

	tflog.Trace(ctx, "CreatePool success")
	return &pool, nil
}

// ExportPool exports (and optionally destroys) a ZFS pool. This is the
// delete operation. It is asynchronous and returns a job ID.
func (c *Client) ExportPool(ctx context.Context, id int, req *PoolExportRequest) error {
	tflog.Trace(ctx, "ExportPool start")

	resp, err := c.Post(ctx, fmt.Sprintf("/pool/id/%d/export", id), req)
	if err != nil {
		return fmt.Errorf("exporting pool %d: %w", id, err)
	}

	if err := c.waitIfJobResponse(ctx, resp, fmt.Sprintf("export pool %d", id)); err != nil {
		return err
	}

	tflog.Trace(ctx, "ExportPool success")
	return nil
}

// GetPoolByName looks up a pool by name.
func (c *Client) GetPoolByName(ctx context.Context, name string) (*Pool, error) {
	tflog.Trace(ctx, "GetPoolByName start")

	pools, err := c.ListPools(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range pools {
		if p.Name == name {
			return &p, nil
		}
	}
	tflog.Trace(ctx, "GetPoolByName success")
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("pool %q not found", name),
	}
}
