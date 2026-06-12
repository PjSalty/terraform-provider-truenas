package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetISCSITargetExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.targetextent.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "target": 1, "extent": 1, "lunid": 0}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetISCSITargetExtent(ctx, 1)
	if err != nil {
		t.Fatalf("GetISCSITargetExtent: %v", err)
	}
	if r.Target != 1 {
		t.Errorf("got %+v", r)
	}
}

func TestGetISCSITargetExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSITargetExtent(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting iSCSI target-extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetISCSITargetExtent_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSITargetExtent(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateISCSITargetExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.targetextent.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "target": 1, "extent": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateISCSITargetExtent(ctx, &types.ISCSITargetExtentCreateRequest{Target: 1, Extent: 2})
	if err != nil {
		t.Fatalf("CreateISCSITargetExtent: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateISCSITargetExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSITargetExtent(ctx, &types.ISCSITargetExtentCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating iSCSI target-extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateISCSITargetExtent_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSITargetExtent(ctx, &types.ISCSITargetExtentCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateISCSITargetExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.targetextent.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "lunid": 5}, nil
	})
	c, _ := ts.NewClient(ctx)
	v := 5
	r, err := c.UpdateISCSITargetExtent(ctx, 2, &types.ISCSITargetExtentUpdateRequest{LunID: &v})
	if err != nil {
		t.Fatalf("UpdateISCSITargetExtent: %v", err)
	}
	if r.LunID != 5 {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateISCSITargetExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSITargetExtent(ctx, 2, &types.ISCSITargetExtentUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI target-extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateISCSITargetExtent_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSITargetExtent(ctx, 2, &types.ISCSITargetExtentUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteISCSITargetExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "iscsi.targetextent.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteISCSITargetExtent(ctx, 2); err != nil {
		t.Fatalf("DeleteISCSITargetExtent: %v", err)
	}
	if !saw {
		t.Error("server did not see iscsi.targetextent.delete")
	}
}

func TestDeleteISCSITargetExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteISCSITargetExtent(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting iSCSI target-extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
