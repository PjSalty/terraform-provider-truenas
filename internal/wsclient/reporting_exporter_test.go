package wsclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetReportingExporter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "reporting.exporters.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1, "name": "graphite-prod", "enabled": true,
			"attributes": map[string]interface{}{"exporter_type": "GRAPHITE", "host": "stats"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	e, err := c.GetReportingExporter(ctx, 1)
	if err != nil {
		t.Fatalf("GetReportingExporter: %v", err)
	}
	if e.Name != "graphite-prod" || !e.Enabled {
		t.Errorf("got %+v", e)
	}
}

func TestGetReportingExporter_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetReportingExporter(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting reporting exporter") {
		t.Errorf("got %v", err)
	}
}

func TestGetReportingExporter_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetReportingExporter(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestCreateReportingExporter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "reporting.exporters.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "name": "graphite", "enabled": true}, nil
	})
	c, _ := ts.NewClient(ctx)
	e, err := c.CreateReportingExporter(ctx, &types.ReportingExporterCreateRequest{
		Name: "graphite", Enabled: true, Attributes: json.RawMessage(`{"exporter_type":"GRAPHITE"}`),
	})
	if err != nil {
		t.Fatalf("CreateReportingExporter: %v", err)
	}
	if e.ID != 9 {
		t.Errorf("got %+v", e)
	}
}

func TestCreateReportingExporter_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateReportingExporter(ctx, &types.ReportingExporterCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating reporting exporter") {
		t.Errorf("got %v", err)
	}
}

func TestCreateReportingExporter_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateReportingExporter(ctx, &types.ReportingExporterCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateReportingExporter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "reporting.exporters.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "name": "graphite-renamed", "enabled": false}, nil
	})
	c, _ := ts.NewClient(ctx)
	enabled := false
	e, err := c.UpdateReportingExporter(ctx, 9, &types.ReportingExporterUpdateRequest{Enabled: &enabled})
	if err != nil {
		t.Fatalf("UpdateReportingExporter: %v", err)
	}
	if e.Enabled {
		t.Errorf("got %+v", e)
	}
}

func TestUpdateReportingExporter_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateReportingExporter(ctx, 9, &types.ReportingExporterUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating reporting exporter") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateReportingExporter_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateReportingExporter(ctx, 9, &types.ReportingExporterUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteReportingExporter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "reporting.exporters.delete" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteReportingExporter(ctx, 9); err != nil {
		t.Errorf("DeleteReportingExporter: %v", err)
	}
}

func TestDeleteReportingExporter_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteReportingExporter(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "deleting reporting exporter") {
		t.Errorf("got %v", err)
	}
}
