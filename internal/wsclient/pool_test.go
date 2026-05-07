package wsclient

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// =============================================================================
// GetPool / ListPools / GetPoolByName
// =============================================================================

func TestGetPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1, "name": "tank", "guid": "abc",
			"path": "/mnt/tank", "status": "ONLINE", "healthy": true,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	pool, err := c.GetPool(ctx, 1)
	if err != nil {
		t.Fatalf("GetPool: %v", err)
	}
	if pool.Name != "tank" || !pool.Healthy {
		t.Errorf("got %+v", pool)
	}
}

func TestGetPool_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPool(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting pool") {
		t.Errorf("got %v", err)
	}
}

func TestGetPool_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPool(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestListPools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{"id": 1, "name": "tank"},
			map[string]interface{}{"id": 2, "name": "fast"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	pools, err := c.ListPools(ctx)
	if err != nil {
		t.Fatalf("ListPools: %v", err)
	}
	if len(pools) != 2 || pools[1].Name != "fast" {
		t.Errorf("got %+v", pools)
	}
}

func TestListPools_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListPools(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing pools") {
		t.Errorf("got %v", err)
	}
}

func TestListPools_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListPools(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetPoolByName_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{map[string]interface{}{"id": 7, "name": "tank"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	pool, err := c.GetPoolByName(ctx, "tank")
	if err != nil {
		t.Fatalf("GetPoolByName: %v", err)
	}
	if pool.ID != 7 {
		t.Errorf("got %+v", pool)
	}
}

func TestGetPoolByName_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPoolByName(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("got %v", err)
	}
}

func TestGetPoolByName_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPoolByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "listing pools") {
		t.Errorf("got %v", err)
	}
}

func TestGetPoolByName_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPoolByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// CreatePool / ExportPool (job-bound)
// =============================================================================

// poolJobServer fakes pool.create or pool.export returning a job ID.
// jobError marks the job FAILED. jobResultPool, when non-nil, is
// returned as the job's result.
// queryResponse is used for the post-create refetch fallback (if
// jobResultPool was not embedded in the job result).
func poolJobServer(t *testing.T, expectMethod string, jobError string, jobResultPool interface{}, queryResponse interface{}) *TestServer {
	t.Helper()
	pollCount := atomic.Int64{}
	const jobID = int64(99)
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case expectMethod:
			return jobID, nil
		case "core.get_jobs":
			pollCount.Add(1)
			state := "SUCCESS"
			if jobError != "" {
				state = "FAILED"
			}
			job := map[string]interface{}{
				"id": jobID, "state": state, "error": jobError,
			}
			if jobResultPool != nil {
				job["result"] = jobResultPool
			}
			return []interface{}{job}, nil
		case "pool.query":
			return queryResponse, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
}

func TestCreatePool_fromJobResult(t *testing.T) {
	// Job returns the placed pool directly.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := poolJobServer(t, "pool.create", "",
		map[string]interface{}{"id": 5, "name": "tank", "healthy": true},
		nil)
	c, _ := ts.NewClient(ctx)
	pool, err := c.CreatePool(ctx, &types.PoolCreateRequest{
		Name: "tank", Topology: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CreatePool: %v", err)
	}
	if pool.ID != 5 || pool.Name != "tank" {
		t.Errorf("got %+v", pool)
	}
}

func TestCreatePool_refetchFallback(t *testing.T) {
	// Job result is null; client falls back to pool.query by name.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := poolJobServer(t, "pool.create", "",
		nil,
		[]interface{}{map[string]interface{}{"id": 5, "name": "tank"}})
	c, _ := ts.NewClient(ctx)
	pool, err := c.CreatePool(ctx, &types.PoolCreateRequest{
		Name: "tank", Topology: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CreatePool: %v", err)
	}
	if pool.ID != 5 {
		t.Errorf("got %+v", pool)
	}
}

func TestCreatePool_jobResultZeroID_refetchFallback(t *testing.T) {
	// Job result decodes but has zero ID -> client falls through to refetch.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := poolJobServer(t, "pool.create", "",
		map[string]interface{}{"id": 0, "name": ""}, // garbage
		[]interface{}{map[string]interface{}{"id": 8, "name": "tank"}})
	c, _ := ts.NewClient(ctx)
	pool, err := c.CreatePool(ctx, &types.PoolCreateRequest{
		Name: "tank", Topology: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CreatePool: %v", err)
	}
	if pool.ID != 8 {
		t.Errorf("got %+v", pool)
	}
}

func TestCreatePool_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := poolJobServer(t, "pool.create", "vdev assembly failed", nil, nil)
	c, _ := ts.NewClient(ctx)
	_, err := c.CreatePool(ctx, &types.PoolCreateRequest{Name: "bad", Topology: json.RawMessage(`{}`)})
	if err == nil || !strings.Contains(err.Error(), "vdev assembly failed") {
		t.Errorf("got %v", err)
	}
}

func TestCreatePool_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "bad topology"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreatePool(ctx, &types.PoolCreateRequest{Name: "x", Topology: json.RawMessage(`{}`)})
	if err == nil || !strings.Contains(err.Error(), "creating pool") {
		t.Errorf("got %v", err)
	}
}

func TestExportPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := poolJobServer(t, "pool.export", "", true, nil)
	c, _ := ts.NewClient(ctx)
	if err := c.ExportPool(ctx, 1, &types.PoolExportRequest{Destroy: true}); err != nil {
		t.Errorf("ExportPool: %v", err)
	}
}

func TestExportPool_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := poolJobServer(t, "pool.export", "pool busy", nil, nil)
	c, _ := ts.NewClient(ctx)
	err := c.ExportPool(ctx, 1, &types.PoolExportRequest{Destroy: true})
	if err == nil || !strings.Contains(err.Error(), "pool busy") {
		t.Errorf("got %v", err)
	}
}

func TestExportPool_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.ExportPool(ctx, 1, &types.PoolExportRequest{})
	if err == nil || !strings.Contains(err.Error(), "exporting pool") {
		t.Errorf("got %v", err)
	}
}
