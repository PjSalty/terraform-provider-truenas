package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetISCSITarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.target.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "tgt", "mode": "ISCSI"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetISCSITarget(ctx, 1)
	if err != nil {
		t.Fatalf("GetISCSITarget: %v", err)
	}
	if r.Name != "tgt" {
		t.Errorf("got %+v", r)
	}
}

func TestGetISCSITarget_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSITarget(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting iSCSI target") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetISCSITarget_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSITarget(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateISCSITarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.target.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "tgt2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateISCSITarget(ctx, &types.ISCSITargetCreateRequest{Name: "tgt2"})
	if err != nil {
		t.Fatalf("CreateISCSITarget: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateISCSITarget_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSITarget(ctx, &types.ISCSITargetCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating iSCSI target") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateISCSITarget_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSITarget(ctx, &types.ISCSITargetCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateISCSITarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.target.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "alias": "new"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateISCSITarget(ctx, 2, &types.ISCSITargetUpdateRequest{Alias: "new"})
	if err != nil {
		t.Fatalf("UpdateISCSITarget: %v", err)
	}
	if r.Alias != "new" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateISCSITarget_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSITarget(ctx, 2, &types.ISCSITargetUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI target") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateISCSITarget_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSITarget(ctx, 2, &types.ISCSITargetUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteISCSITarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "iscsi.target.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteISCSITarget(ctx, 2); err != nil {
		t.Fatalf("DeleteISCSITarget: %v", err)
	}
	if !saw {
		t.Error("server did not see iscsi.target.delete")
	}
}

func TestDeleteISCSITarget_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteISCSITarget(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting iSCSI target") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
