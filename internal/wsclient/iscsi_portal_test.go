package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetISCSIPortal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.portal.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "tag": 1}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetISCSIPortal(ctx, 1)
	if err != nil {
		t.Fatalf("GetISCSIPortal: %v", err)
	}
	if r.Tag != 1 {
		t.Errorf("got %+v", r)
	}
}

func TestGetISCSIPortal_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIPortal(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting iSCSI portal") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetISCSIPortal_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIPortal(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateISCSIPortal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.portal.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "tag": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateISCSIPortal(ctx, &types.ISCSIPortalCreateRequest{
		Listen: []types.ISCSIPortalListen{{IP: "0.0.0.0"}},
	})
	if err != nil {
		t.Fatalf("CreateISCSIPortal: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateISCSIPortal_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIPortal(ctx, &types.ISCSIPortalCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating iSCSI portal") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateISCSIPortal_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIPortal(ctx, &types.ISCSIPortalCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateISCSIPortal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.portal.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "comment": "updated"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateISCSIPortal(ctx, 2, &types.ISCSIPortalUpdateRequest{Comment: "updated"})
	if err != nil {
		t.Fatalf("UpdateISCSIPortal: %v", err)
	}
	if r.Comment != "updated" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateISCSIPortal_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIPortal(ctx, 2, &types.ISCSIPortalUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI portal") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateISCSIPortal_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIPortal(ctx, 2, &types.ISCSIPortalUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteISCSIPortal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "iscsi.portal.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteISCSIPortal(ctx, 2); err != nil {
		t.Fatalf("DeleteISCSIPortal: %v", err)
	}
	if !saw {
		t.Error("server did not see iscsi.portal.delete")
	}
}

func TestDeleteISCSIPortal_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteISCSIPortal(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting iSCSI portal") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
