package wsclient

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// =============================================================================
// GetSystemDataset
// =============================================================================

func TestGetSystemDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "systemdataset.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1, "pool": "tank", "pool_set": true,
			"uuid": "abc", "basename": "tank/.system", "path": "/var/db/system",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetSystemDataset(ctx)
	if err != nil {
		t.Fatalf("GetSystemDataset: %v", err)
	}
	if cfg.Pool != "tank" || !cfg.PoolSet {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetSystemDataset_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSystemDataset(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting system dataset") {
		t.Errorf("got %v", err)
	}
}

func TestGetSystemDataset_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSystemDataset(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// UpdateSystemDataset
// =============================================================================

// systemDatasetJobServer fakes systemdataset.update returning a job ID,
// then either succeeding the job (and answering the follow-up
// systemdataset.config with placedConfig) or failing it.
func systemDatasetJobServer(t *testing.T, jobError string, placedConfig interface{}) *TestServer {
	t.Helper()
	pollCount := atomic.Int64{}
	const jobID = int64(11)
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "systemdataset.update":
			return jobID, nil
		case "core.get_jobs":
			pollCount.Add(1)
			state := "SUCCESS"
			if jobError != "" {
				state = "FAILED"
			}
			job := map[string]interface{}{
				"id": jobID, "state": state, "result": nil, "error": jobError,
			}
			return []interface{}{job}, nil
		case "systemdataset.config":
			return placedConfig, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
}

func TestUpdateSystemDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool := "newpool"
	ts := systemDatasetJobServer(t, "",
		map[string]interface{}{
			"id": 1, "pool": "newpool", "pool_set": true,
			"uuid": "z", "basename": "newpool/.system", "path": "/var/db/system",
		})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.UpdateSystemDataset(ctx, &types.SystemDatasetUpdateRequest{Pool: &pool})
	if err != nil {
		t.Fatalf("UpdateSystemDataset: %v", err)
	}
	if cfg.Pool != "newpool" {
		t.Errorf("got %+v", cfg)
	}
}

func TestUpdateSystemDataset_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := systemDatasetJobServer(t, "no such pool", nil)
	c, _ := ts.NewClient(ctx)
	pool := "missing"
	_, err := c.UpdateSystemDataset(ctx, &types.SystemDatasetUpdateRequest{Pool: &pool})
	if err == nil || !strings.Contains(err.Error(), "no such pool") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateSystemDataset_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "bad params"}
	})
	c, _ := ts.NewClient(ctx)
	pool := "x"
	_, err := c.UpdateSystemDataset(ctx, &types.SystemDatasetUpdateRequest{Pool: &pool})
	if err == nil || !strings.Contains(err.Error(), "updating system dataset") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateSystemDataset_followupConfigError(t *testing.T) {
	// Job succeeds but the follow-on systemdataset.config call errors;
	// that error must surface to the caller.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(11)
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "systemdataset.update":
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": jobID, "state": "SUCCESS", "result": nil,
			}}, nil
		case "systemdataset.config":
			return nil, &RPCError{Code: CodeInternalError, Message: "post-job read failed"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	pool := "y"
	_, err := c.UpdateSystemDataset(ctx, &types.SystemDatasetUpdateRequest{Pool: &pool})
	if err == nil || !strings.Contains(err.Error(), "getting system dataset") {
		t.Errorf("got %v", err)
	}
}
