package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for the SCALE catalog singleton:
// catalog.{config, update, sync}.
//
// catalog.sync is a job server-side (it pulls + parses helm chart
// manifests for every train); CallJob waits for terminal state.
const catalogPollInterval = 1 * time.Second

// GetCatalog retrieves the singleton catalog configuration.
func (c *Client) GetCatalog(ctx context.Context) (*types.Catalog, error) {
	tflog.Trace(ctx, "GetCatalog (ws) start")

	result, err := c.Call(ctx, "catalog.config", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting catalog: %w", err)
	}

	var cat types.Catalog
	if err := json.Unmarshal(result, &cat); err != nil {
		return nil, fmt.Errorf("parsing catalog response: %w", err)
	}

	tflog.Trace(ctx, "GetCatalog (ws) success")
	return &cat, nil
}

// UpdateCatalog updates the singleton catalog configuration.
func (c *Client) UpdateCatalog(ctx context.Context, req *types.CatalogUpdateRequest) (*types.Catalog, error) {
	tflog.Trace(ctx, "UpdateCatalog (ws) start")

	result, err := c.Call(ctx, "catalog.update",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating catalog: %w", err)
	}

	var cat types.Catalog
	if err := json.Unmarshal(result, &cat); err != nil {
		return nil, fmt.Errorf("parsing catalog update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCatalog (ws) success")
	return &cat, nil
}

// SyncCatalog triggers a catalog sync (async job server-side).
func (c *Client) SyncCatalog(ctx context.Context) error {
	tflog.Trace(ctx, "SyncCatalog (ws) start")

	if _, err := c.CallJob(ctx, "catalog.sync", nil,
		CallOptions{Job: true, Idempotent: false},
		catalogPollInterval); err != nil {
		return fmt.Errorf("triggering catalog sync: %w", err)
	}

	tflog.Trace(ctx, "SyncCatalog (ws) success")
	return nil
}
