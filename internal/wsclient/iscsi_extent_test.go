package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetISCSIExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.extent.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "ex", "type": "DISK", "blocksize": 512}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetISCSIExtent(ctx, 1)
	if err != nil {
		t.Fatalf("GetISCSIExtent: %v", err)
	}
	if r.Name != "ex" {
		t.Errorf("got %+v", r)
	}
}

func TestGetISCSIExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIExtent(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting iSCSI extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetISCSIExtent_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIExtent(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateISCSIExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.extent.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "n2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateISCSIExtent(ctx, &types.ISCSIExtentCreateRequest{Name: "n2", Type: "DISK", Blocksize: 512})
	if err != nil {
		t.Fatalf("CreateISCSIExtent: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateISCSIExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIExtent(ctx, &types.ISCSIExtentCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating iSCSI extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateISCSIExtent_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIExtent(ctx, &types.ISCSIExtentCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateISCSIExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.extent.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "comment": "updated"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateISCSIExtent(ctx, 2, &types.ISCSIExtentUpdateRequest{Comment: "updated"})
	if err != nil {
		t.Fatalf("UpdateISCSIExtent: %v", err)
	}
	if r.Comment != "updated" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateISCSIExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIExtent(ctx, 2, &types.ISCSIExtentUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateISCSIExtent_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIExtent(ctx, 2, &types.ISCSIExtentUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteISCSIExtent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "iscsi.extent.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteISCSIExtent(ctx, 2); err != nil {
		t.Fatalf("DeleteISCSIExtent: %v", err)
	}
	if !saw {
		t.Error("server did not see iscsi.extent.delete")
	}
}

func TestDeleteISCSIExtent_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteISCSIExtent(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting iSCSI extent") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
