package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":   "tank/data",
			"name": "tank/data",
			"pool": "tank",
			"type": "FILESYSTEM",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.GetDataset(ctx, "tank/data")
	if err != nil {
		t.Fatalf("GetDataset: %v", err)
	}
	if d.ID != "tank/data" {
		t.Errorf("got %+v", d)
	}
}

func TestGetDataset_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetDataset(ctx, "tank/x")
	if err == nil || !strings.Contains(err.Error(), "getting dataset") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetDataset_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetDataset(ctx, "tank/x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestListDatasets(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{"id": "tank", "name": "tank", "pool": "tank"},
			map[string]interface{}{"id": "tank/data", "name": "tank/data", "pool": "tank"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	ds, err := c.ListDatasets(ctx)
	if err != nil {
		t.Fatalf("ListDatasets: %v", err)
	}
	if len(ds) != 2 {
		t.Errorf("got %+v", ds)
	}
}

func TestListDatasets_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListDatasets(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing datasets") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListDatasets_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListDatasets(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": "tank/new", "name": "tank/new", "pool": "tank"}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.CreateDataset(ctx, &types.DatasetCreateRequest{Name: "tank/new"})
	if err != nil {
		t.Fatalf("CreateDataset: %v", err)
	}
	if d.ID != "tank/new" {
		t.Errorf("got %+v", d)
	}
}

func TestCreateDataset_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateDataset(ctx, &types.DatasetCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating dataset") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateDataset_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateDataset(ctx, &types.DatasetCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": "tank/x", "name": "tank/x"}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.UpdateDataset(ctx, "tank/x", &types.DatasetUpdateRequest{Compression: "LZ4"})
	if err != nil {
		t.Fatalf("UpdateDataset: %v", err)
	}
	if d.ID != "tank/x" {
		t.Errorf("got %+v", d)
	}
}

func TestUpdateDataset_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateDataset(ctx, "tank/x", &types.DatasetUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating dataset") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateDataset_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateDataset(ctx, "tank/x", &types.DatasetUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteDataset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "pool.dataset.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteDataset(ctx, "tank/x"); err != nil {
		t.Fatalf("DeleteDataset: %v", err)
	}
	if !saw {
		t.Error("server did not see pool.dataset.delete")
	}
}

func TestDeleteDataset_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteDataset(ctx, "tank/x")
	if err == nil || !strings.Contains(err.Error(), "deleting dataset") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
