package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for SCALE applications:
// app.{query, get_instance, create, update, delete}.
//
// app.create / app.update / app.delete are asynchronous on the server
// (image pull + helm install + persistent-volume provisioning can run
// for many seconds). CallJob waits for terminal state. After job
// completion, this client refetches the app via app.get_instance to
// return up-to-date placement data (state, version, metadata) — the
// job's `result` field is not contractually guaranteed to contain the
// final shape across SCALE point releases.
const appPollInterval = 1 * time.Second

// ListApps retrieves all deployed apps.
func (c *Client) ListApps(ctx context.Context) ([]types.App, error) {
	tflog.Trace(ctx, "ListApps (ws) start")

	result, err := c.Call(ctx, "app.query", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing apps: %w", err)
	}

	var apps []types.App
	if err := json.Unmarshal(result, &apps); err != nil {
		return nil, fmt.Errorf("parsing apps list response: %w", err)
	}

	tflog.Trace(ctx, "ListApps (ws) success")
	return apps, nil
}

// GetApp retrieves a deployed app by its string ID (app name).
func (c *Client) GetApp(ctx context.Context, id string) (*types.App, error) {
	tflog.Trace(ctx, "GetApp (ws) start")

	result, err := c.Call(ctx, "app.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting app %q: %w", id, err)
	}

	var app types.App
	if err := json.Unmarshal(result, &app); err != nil {
		return nil, fmt.Errorf("parsing app response: %w", err)
	}

	tflog.Trace(ctx, "GetApp (ws) success")
	return &app, nil
}

// CreateApp installs a new app. The underlying app.create RPC is a
// job (image pull + helm install). After CallJob returns, this method
// fetches the placed app via app.get_instance — the job's result
// shape is not stable across SCALE releases, so refetching keeps the
// returned struct schema-true regardless of server version.
func (c *Client) CreateApp(ctx context.Context, req *types.AppCreateRequest) (*types.App, error) {
	tflog.Trace(ctx, "CreateApp (ws) start")

	_, err := c.CallJob(ctx, "app.create",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: false},
		appPollInterval)
	if err != nil {
		return nil, fmt.Errorf("creating app %q: %w", req.AppName, err)
	}

	tflog.Trace(ctx, "CreateApp (ws) success")
	return c.GetApp(ctx, req.AppName)
}

// UpdateApp updates an existing app. Same job-and-refetch pattern as
// CreateApp — the helm upgrade runs server-side.
func (c *Client) UpdateApp(ctx context.Context, id string, req *types.AppUpdateRequest) (*types.App, error) {
	tflog.Trace(ctx, "UpdateApp (ws) start")

	_, err := c.CallJob(ctx, "app.update",
		[]interface{}{id, req},
		CallOptions{Job: true, Idempotent: false},
		appPollInterval)
	if err != nil {
		return nil, fmt.Errorf("updating app %q: %w", id, err)
	}

	tflog.Trace(ctx, "UpdateApp (ws) success")
	return c.GetApp(ctx, id)
}

// DeleteApp deletes an app, optionally reaping its container images
// and ix-volume PVs as well. Async server-side.
func (c *Client) DeleteApp(ctx context.Context, id string, req *types.AppDeleteRequest) error {
	tflog.Trace(ctx, "DeleteApp (ws) start")

	_, err := c.CallJob(ctx, "app.delete",
		[]interface{}{id, req},
		CallOptions{Job: true, Idempotent: false},
		appPollInterval)
	if err != nil {
		return fmt.Errorf("deleting app %q: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteApp (ws) success")
	return nil
}
