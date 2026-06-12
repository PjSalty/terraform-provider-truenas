package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for cron jobs: cronjob.{...}

// ListCronJobs retrieves all cron jobs.
func (c *Client) ListCronJobs(ctx context.Context) ([]types.CronJob, error) {
	tflog.Trace(ctx, "ListCronJobs (ws) start")

	result, err := c.Call(ctx, "cronjob.query", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing cron jobs: %w", err)
	}

	var jobs []types.CronJob
	if err := json.Unmarshal(result, &jobs); err != nil {
		return nil, fmt.Errorf("parsing cron jobs list: %w", err)
	}

	tflog.Trace(ctx, "ListCronJobs (ws) success")
	return jobs, nil
}

// GetCronJob retrieves a cron job by ID.
func (c *Client) GetCronJob(ctx context.Context, id int) (*types.CronJob, error) {
	tflog.Trace(ctx, "GetCronJob (ws) start")

	result, err := c.Call(ctx, "cronjob.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting cron job %d: %w", id, err)
	}

	var job types.CronJob
	if err := json.Unmarshal(result, &job); err != nil {
		return nil, fmt.Errorf("parsing cron job response: %w", err)
	}

	tflog.Trace(ctx, "GetCronJob (ws) success")
	return &job, nil
}

// CreateCronJob creates a new cron job.
func (c *Client) CreateCronJob(ctx context.Context, req *types.CronJobCreateRequest) (*types.CronJob, error) {
	tflog.Trace(ctx, "CreateCronJob (ws) start")

	result, err := c.Call(ctx, "cronjob.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating cron job: %w", err)
	}

	var job types.CronJob
	if err := json.Unmarshal(result, &job); err != nil {
		return nil, fmt.Errorf("parsing cron job create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCronJob (ws) success")
	return &job, nil
}

// UpdateCronJob updates an existing cron job.
func (c *Client) UpdateCronJob(ctx context.Context, id int, req *types.CronJobUpdateRequest) (*types.CronJob, error) {
	tflog.Trace(ctx, "UpdateCronJob (ws) start")

	result, err := c.Call(ctx, "cronjob.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating cron job %d: %w", id, err)
	}

	var job types.CronJob
	if err := json.Unmarshal(result, &job); err != nil {
		return nil, fmt.Errorf("parsing cron job update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCronJob (ws) success")
	return &job, nil
}

// DeleteCronJob deletes a cron job by ID.
func (c *Client) DeleteCronJob(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCronJob (ws) start")

	if _, err := c.Call(ctx, "cronjob.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting cron job %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteCronJob (ws) success")
	return nil
}
