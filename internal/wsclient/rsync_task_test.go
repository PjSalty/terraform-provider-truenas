package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetRsyncTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "rsynctask.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "path": "/mnt/x", "user": "root"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetRsyncTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetRsyncTask: %v", err)
	}
	if r.User != "root" {
		t.Errorf("got %+v", r)
	}
}

func TestGetRsyncTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetRsyncTask(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting rsync task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetRsyncTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetRsyncTask(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateRsyncTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "rsynctask.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "path": "/mnt/y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateRsyncTask(ctx, &types.RsyncTaskCreateRequest{Path: "/mnt/y", User: "root"})
	if err != nil {
		t.Fatalf("CreateRsyncTask: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateRsyncTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateRsyncTask(ctx, &types.RsyncTaskCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating rsync task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateRsyncTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateRsyncTask(ctx, &types.RsyncTaskCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateRsyncTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "rsynctask.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "user": "u2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateRsyncTask(ctx, 2, &types.RsyncTaskUpdateRequest{User: "u2"})
	if err != nil {
		t.Fatalf("UpdateRsyncTask: %v", err)
	}
	if r.User != "u2" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateRsyncTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateRsyncTask(ctx, 2, &types.RsyncTaskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating rsync task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateRsyncTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateRsyncTask(ctx, 2, &types.RsyncTaskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteRsyncTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "rsynctask.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteRsyncTask(ctx, 2); err != nil {
		t.Fatalf("DeleteRsyncTask: %v", err)
	}
	if !saw {
		t.Error("server did not see rsynctask.delete")
	}
}

func TestDeleteRsyncTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteRsyncTask(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting rsync task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
