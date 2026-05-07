package wsclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetCloudBackup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloud_backup.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "path": "/mnt/x", "keep_last": 5}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetCloudBackup(ctx, 1)
	if err != nil {
		t.Fatalf("GetCloudBackup: %v", err)
	}
	if r.KeepLast != 5 {
		t.Errorf("got %+v", r)
	}
}

func TestGetCloudBackup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudBackup(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting cloud backup") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetCloudBackup_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudBackup(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateCloudBackup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloud_backup.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "path": "/mnt/y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateCloudBackup(ctx, &types.CloudBackupCreateRequest{Path: "/mnt/y", Attributes: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("CreateCloudBackup: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateCloudBackup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCloudBackup(ctx, &types.CloudBackupCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating cloud backup") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateCloudBackup_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCloudBackup(ctx, &types.CloudBackupCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateCloudBackup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloud_backup.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "keep_last": 7}, nil
	})
	c, _ := ts.NewClient(ctx)
	keep := 7
	r, err := c.UpdateCloudBackup(ctx, 2, &types.CloudBackupUpdateRequest{KeepLast: &keep})
	if err != nil {
		t.Fatalf("UpdateCloudBackup: %v", err)
	}
	if r.KeepLast != 7 {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateCloudBackup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCloudBackup(ctx, 2, &types.CloudBackupUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating cloud backup") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateCloudBackup_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCloudBackup(ctx, 2, &types.CloudBackupUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteCloudBackup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "cloud_backup.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteCloudBackup(ctx, 2); err != nil {
		t.Fatalf("DeleteCloudBackup: %v", err)
	}
	if !saw {
		t.Error("server did not see cloud_backup.delete")
	}
}

func TestDeleteCloudBackup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteCloudBackup(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting cloud backup") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
