package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetCloudSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "path": "/mnt/x"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetCloudSync(ctx, 1)
	if err != nil {
		t.Fatalf("GetCloudSync: %v", err)
	}
	if r.Path != "/mnt/x" {
		t.Errorf("got %+v", r)
	}
}

func TestGetCloudSync_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudSync(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting cloud sync") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetCloudSync_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudSync(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateCloudSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "path": "/mnt/y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateCloudSync(ctx, &types.CloudSyncCreateRequest{Path: "/mnt/y"})
	if err != nil {
		t.Fatalf("CreateCloudSync: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateCloudSync_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCloudSync(ctx, &types.CloudSyncCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating cloud sync") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateCloudSync_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCloudSync(ctx, &types.CloudSyncCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateCloudSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "description": "u"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateCloudSync(ctx, 2, &types.CloudSyncUpdateRequest{Description: "u"})
	if err != nil {
		t.Fatalf("UpdateCloudSync: %v", err)
	}
	if r.Description != "u" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateCloudSync_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCloudSync(ctx, 2, &types.CloudSyncUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating cloud sync") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateCloudSync_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCloudSync(ctx, 2, &types.CloudSyncUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteCloudSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "cloudsync.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteCloudSync(ctx, 2); err != nil {
		t.Fatalf("DeleteCloudSync: %v", err)
	}
	if !saw {
		t.Error("server did not see cloudsync.delete")
	}
}

func TestDeleteCloudSync_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteCloudSync(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting cloud sync") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
