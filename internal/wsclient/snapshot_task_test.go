package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetSnapshotTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.snapshottask.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "dataset": "tank/x", "lifetime_value": 1, "lifetime_unit": "WEEK"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetSnapshotTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetSnapshotTask: %v", err)
	}
	if r.Dataset != "tank/x" {
		t.Errorf("got %+v", r)
	}
}

func TestGetSnapshotTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSnapshotTask(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting snapshot task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetSnapshotTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSnapshotTask(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateSnapshotTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.snapshottask.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "dataset": "tank/y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateSnapshotTask(ctx, &types.SnapshotTaskCreateRequest{Dataset: "tank/y"})
	if err != nil {
		t.Fatalf("CreateSnapshotTask: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateSnapshotTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateSnapshotTask(ctx, &types.SnapshotTaskCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating snapshot task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateSnapshotTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateSnapshotTask(ctx, &types.SnapshotTaskCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateSnapshotTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.snapshottask.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "dataset": "tank/z"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateSnapshotTask(ctx, 2, &types.SnapshotTaskUpdateRequest{Dataset: "tank/z"})
	if err != nil {
		t.Fatalf("UpdateSnapshotTask: %v", err)
	}
	if r.Dataset != "tank/z" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateSnapshotTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSnapshotTask(ctx, 2, &types.SnapshotTaskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating snapshot task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateSnapshotTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSnapshotTask(ctx, 2, &types.SnapshotTaskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteSnapshotTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "pool.snapshottask.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteSnapshotTask(ctx, 2); err != nil {
		t.Fatalf("DeleteSnapshotTask: %v", err)
	}
	if !saw {
		t.Error("server did not see pool.snapshottask.delete")
	}
}

func TestDeleteSnapshotTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteSnapshotTask(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting snapshot task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
