package wsclient

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetCatalog(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "catalog.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": "TRUENAS", "label": "TRUENAS",
			"preferred_trains": []interface{}{"stable"},
			"location":         "/var/lib/catalog",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cat, err := c.GetCatalog(ctx)
	if err != nil {
		t.Fatalf("GetCatalog: %v", err)
	}
	if cat.ID != "TRUENAS" || len(cat.PreferredTrains) != 1 {
		t.Errorf("got %+v", cat)
	}
}

func TestGetCatalog_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCatalog(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting catalog") {
		t.Errorf("got %v", err)
	}
}

func TestGetCatalog_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCatalog(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateCatalog(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "catalog.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": "TRUENAS", "label": "TRUENAS",
			"preferred_trains": []interface{}{"stable", "community"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	trains := []string{"stable", "community"}
	cat, err := c.UpdateCatalog(ctx, &types.CatalogUpdateRequest{PreferredTrains: &trains})
	if err != nil {
		t.Fatalf("UpdateCatalog: %v", err)
	}
	if len(cat.PreferredTrains) != 2 {
		t.Errorf("got %+v", cat)
	}
}

func TestUpdateCatalog_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCatalog(ctx, &types.CatalogUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating catalog") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateCatalog_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCatalog(ctx, &types.CatalogUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestSyncCatalog(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pollCount := atomic.Int64{}
	const jobID = int64(60)
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "catalog.sync":
			return jobID, nil
		case "core.get_jobs":
			pollCount.Add(1)
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.SyncCatalog(ctx); err != nil {
		t.Errorf("SyncCatalog: %v", err)
	}
}

func TestSyncCatalog_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(60)
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "catalog.sync":
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "FAILED", "error": "fetch failed"}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	err := c.SyncCatalog(ctx)
	if err == nil || !strings.Contains(err.Error(), "fetch failed") {
		t.Errorf("got %v", err)
	}
}

func TestSyncCatalog_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.SyncCatalog(ctx)
	if err == nil || !strings.Contains(err.Error(), "triggering catalog sync") {
		t.Errorf("got %v", err)
	}
}
