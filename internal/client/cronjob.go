package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// CronJob, CronJobCreateRequest, CronJobUpdateRequest moved to
// internal/types/cronjob.go in the v2.0 transport-migration prep.
type (
	CronJob              = types.CronJob
	CronJobCreateRequest = types.CronJobCreateRequest
	CronJobUpdateRequest = types.CronJobUpdateRequest
)

// ListCronJobs retrieves all cron jobs.
func (c *Client) ListCronJobs(ctx context.Context) ([]types.CronJob, error) {
	tflog.Trace(ctx, "ListCronJobs start")

	resp, err := c.Get(ctx, "/cronjob")
	if err != nil {
		return nil, fmt.Errorf("listing cron jobs: %w", err)
	}

	var jobs []types.CronJob
	if err := json.Unmarshal(resp, &jobs); err != nil {
		return nil, fmt.Errorf("parsing cron jobs list: %w", err)
	}

	tflog.Trace(ctx, "ListCronJobs success")
	return jobs, nil
}

// GetCronJob retrieves a cron job by ID.
func (c *Client) GetCronJob(ctx context.Context, id int) (*types.CronJob, error) {
	tflog.Trace(ctx, "GetCronJob start")

	resp, err := c.Get(ctx, fmt.Sprintf("/cronjob/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting cron job %d: %w", id, err)
	}

	var job types.CronJob
	if err := json.Unmarshal(resp, &job); err != nil {
		return nil, fmt.Errorf("parsing cron job response: %w", err)
	}

	tflog.Trace(ctx, "GetCronJob success")
	return &job, nil
}

// CreateCronJob creates a new cron job.
func (c *Client) CreateCronJob(ctx context.Context, req *types.CronJobCreateRequest) (*types.CronJob, error) {
	tflog.Trace(ctx, "CreateCronJob start")

	resp, err := c.Post(ctx, "/cronjob", req)
	if err != nil {
		return nil, fmt.Errorf("creating cron job: %w", err)
	}

	var job types.CronJob
	if err := json.Unmarshal(resp, &job); err != nil {
		return nil, fmt.Errorf("parsing cron job create response: %w", err)
	}

	tflog.Trace(ctx, "CreateCronJob success")
	return &job, nil
}

// UpdateCronJob updates an existing cron job.
func (c *Client) UpdateCronJob(ctx context.Context, id int, req *types.CronJobUpdateRequest) (*types.CronJob, error) {
	tflog.Trace(ctx, "UpdateCronJob start")

	resp, err := c.Put(ctx, fmt.Sprintf("/cronjob/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating cron job %d: %w", id, err)
	}

	var job types.CronJob
	if err := json.Unmarshal(resp, &job); err != nil {
		return nil, fmt.Errorf("parsing cron job update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateCronJob success")
	return &job, nil
}

// DeleteCronJob deletes a cron job.
func (c *Client) DeleteCronJob(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCronJob start")

	_, err := c.Delete(ctx, fmt.Sprintf("/cronjob/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting cron job %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteCronJob success")
	return nil
}
