package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for the system dataset singleton:
// systemdataset.{config, update}.
//
// systemdataset.update is asynchronous on the server (pool moves can
// take many seconds — actual data is copied between pools). CallJob
// waits for terminal state, then a follow-on systemdataset.config
// fetches the final placed configuration.
const systemDatasetPollInterval = 1 * time.Second

// GetSystemDataset retrieves the current system dataset configuration.
func (c *Client) GetSystemDataset(ctx context.Context) (*types.SystemDataset, error) {
	tflog.Trace(ctx, "GetSystemDataset (ws) start")

	result, err := c.Call(ctx, "systemdataset.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting system dataset: %w", err)
	}

	var cfg types.SystemDataset
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing system dataset response: %w", err)
	}

	tflog.Trace(ctx, "GetSystemDataset (ws) success")
	return &cfg, nil
}

// UpdateSystemDataset updates the system dataset configuration. The
// underlying systemdataset.update RPC is run as a job server-side
// because moving the dataset to a different pool requires copying
// data; CallJob waits for the job to finish, then this method fetches
// the post-move configuration via systemdataset.config so the caller
// gets the actual placed state (including any server-derived fields
// like uuid/path).
func (c *Client) UpdateSystemDataset(ctx context.Context, req *types.SystemDatasetUpdateRequest) (*types.SystemDataset, error) {
	tflog.Trace(ctx, "UpdateSystemDataset (ws) start")

	_, err := c.CallJob(ctx, "systemdataset.update",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: false},
		systemDatasetPollInterval)
	if err != nil {
		return nil, fmt.Errorf("updating system dataset: %w", err)
	}

	tflog.Trace(ctx, "UpdateSystemDataset (ws) success")
	return c.GetSystemDataset(ctx)
}
