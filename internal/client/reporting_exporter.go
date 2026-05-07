package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ReportingExporter, ReportingExporterCreateRequest,
// ReportingExporterUpdateRequest moved to
// internal/types/reporting_exporter.go in the v2.0
// transport-migration prep.
type (
	ReportingExporter              = types.ReportingExporter
	ReportingExporterCreateRequest = types.ReportingExporterCreateRequest
	ReportingExporterUpdateRequest = types.ReportingExporterUpdateRequest
)

// GetReportingExporter retrieves an exporter by ID.
func (c *Client) GetReportingExporter(ctx context.Context, id int) (*ReportingExporter, error) {
	tflog.Trace(ctx, "GetReportingExporter start")

	resp, err := c.Get(ctx, fmt.Sprintf("/reporting/exporters/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting reporting exporter %d: %w", id, err)
	}

	var e ReportingExporter
	if err := json.Unmarshal(resp, &e); err != nil {
		return nil, fmt.Errorf("parsing reporting exporter response: %w", err)
	}

	tflog.Trace(ctx, "GetReportingExporter success")
	return &e, nil
}

// CreateReportingExporter creates an exporter.
func (c *Client) CreateReportingExporter(ctx context.Context, req *ReportingExporterCreateRequest) (*ReportingExporter, error) {
	tflog.Trace(ctx, "CreateReportingExporter start")

	resp, err := c.Post(ctx, "/reporting/exporters", req)
	if err != nil {
		return nil, fmt.Errorf("creating reporting exporter: %w", err)
	}

	var e ReportingExporter
	if err := json.Unmarshal(resp, &e); err != nil {
		return nil, fmt.Errorf("parsing reporting exporter create response: %w", err)
	}

	tflog.Trace(ctx, "CreateReportingExporter success")
	return &e, nil
}

// UpdateReportingExporter updates an exporter by ID.
func (c *Client) UpdateReportingExporter(ctx context.Context, id int, req *ReportingExporterUpdateRequest) (*ReportingExporter, error) {
	tflog.Trace(ctx, "UpdateReportingExporter start")

	resp, err := c.Put(ctx, fmt.Sprintf("/reporting/exporters/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating reporting exporter %d: %w", id, err)
	}

	var e ReportingExporter
	if err := json.Unmarshal(resp, &e); err != nil {
		return nil, fmt.Errorf("parsing reporting exporter update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateReportingExporter success")
	return &e, nil
}

// DeleteReportingExporter deletes an exporter by ID.
func (c *Client) DeleteReportingExporter(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteReportingExporter start")

	_, err := c.Delete(ctx, fmt.Sprintf("/reporting/exporters/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting reporting exporter %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteReportingExporter success")
	return nil
}
