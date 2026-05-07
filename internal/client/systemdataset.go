package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- System Dataset API ---
//
// The system dataset is a singleton configuration describing which pool
// hosts TrueNAS's internal system data (samba, reports, syslog, ...).
// PUT /systemdataset is asynchronous and returns a job ID.

// SystemDataset, SystemDatasetUpdateRequest moved to
// internal/types/systemdataset.go in the v2.0 transport-migration prep.
type (
	SystemDataset              = types.SystemDataset
	SystemDatasetUpdateRequest = types.SystemDatasetUpdateRequest
)

// GetSystemDataset retrieves the current system dataset configuration.
func (c *Client) GetSystemDataset(ctx context.Context) (*SystemDataset, error) {
	tflog.Trace(ctx, "GetSystemDataset start")

	resp, err := c.Get(ctx, "/systemdataset")
	if err != nil {
		return nil, fmt.Errorf("getting system dataset: %w", err)
	}
	var cfg SystemDataset
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing system dataset response: %w", err)
	}
	tflog.Trace(ctx, "GetSystemDataset success")
	return &cfg, nil
}

// UpdateSystemDataset updates the system dataset configuration. This is an
// async job (pool moves can take time) so we wait for completion.
func (c *Client) UpdateSystemDataset(ctx context.Context, req *SystemDatasetUpdateRequest) (*SystemDataset, error) {
	tflog.Trace(ctx, "UpdateSystemDataset start")

	resp, err := c.Put(ctx, "/systemdataset", req)
	if err != nil {
		return nil, fmt.Errorf("updating system dataset: %w", err)
	}

	// PUT /systemdataset returns a job ID; wait for completion, then re-read.
	var jobID int
	if err := json.Unmarshal(resp, &jobID); err != nil {
		// Some versions may return the config directly; try that fallback.
		var cfg SystemDataset
		if err2 := json.Unmarshal(resp, &cfg); err2 == nil {
			return &cfg, nil
		}
		return nil, fmt.Errorf("parsing system dataset update response: %w", err)
	}

	if _, err := c.WaitForJob(ctx, jobID); err != nil {
		return nil, fmt.Errorf("waiting for system dataset update: %w", err)
	}

	tflog.Trace(ctx, "UpdateSystemDataset success")
	return c.GetSystemDataset(ctx)
}
