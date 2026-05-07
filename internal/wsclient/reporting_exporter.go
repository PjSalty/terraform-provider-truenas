package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for reporting exporters:
// reporting.exporters.{get_instance, create, update, delete}.

// GetReportingExporter retrieves an exporter by ID.
func (c *Client) GetReportingExporter(ctx context.Context, id int) (*types.ReportingExporter, error) {
	tflog.Trace(ctx, "GetReportingExporter (ws) start")

	result, err := c.Call(ctx, "reporting.exporters.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting reporting exporter %d: %w", id, err)
	}

	var e types.ReportingExporter
	if err := json.Unmarshal(result, &e); err != nil {
		return nil, fmt.Errorf("parsing reporting exporter response: %w", err)
	}

	tflog.Trace(ctx, "GetReportingExporter (ws) success")
	return &e, nil
}

// CreateReportingExporter creates an exporter.
func (c *Client) CreateReportingExporter(ctx context.Context, req *types.ReportingExporterCreateRequest) (*types.ReportingExporter, error) {
	tflog.Trace(ctx, "CreateReportingExporter (ws) start")

	result, err := c.Call(ctx, "reporting.exporters.create",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("creating reporting exporter: %w", err)
	}

	var e types.ReportingExporter
	if err := json.Unmarshal(result, &e); err != nil {
		return nil, fmt.Errorf("parsing reporting exporter create response: %w", err)
	}

	tflog.Trace(ctx, "CreateReportingExporter (ws) success")
	return &e, nil
}

// UpdateReportingExporter updates an exporter by ID.
func (c *Client) UpdateReportingExporter(ctx context.Context, id int, req *types.ReportingExporterUpdateRequest) (*types.ReportingExporter, error) {
	tflog.Trace(ctx, "UpdateReportingExporter (ws) start")

	result, err := c.Call(ctx, "reporting.exporters.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating reporting exporter %d: %w", id, err)
	}

	var e types.ReportingExporter
	if err := json.Unmarshal(result, &e); err != nil {
		return nil, fmt.Errorf("parsing reporting exporter update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateReportingExporter (ws) success")
	return &e, nil
}

// DeleteReportingExporter deletes an exporter by ID.
func (c *Client) DeleteReportingExporter(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteReportingExporter (ws) start")

	if _, err := c.Call(ctx, "reporting.exporters.delete",
		[]interface{}{id},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("deleting reporting exporter %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteReportingExporter (ws) success")
	return nil
}
