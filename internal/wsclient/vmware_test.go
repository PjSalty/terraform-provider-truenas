package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetVMware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vmware.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "hostname": "vc.local"}, nil
	})
	c, _ := ts.NewClient(ctx)
	v, err := c.GetVMware(ctx, 1)
	if err != nil {
		t.Fatalf("GetVMware: %v", err)
	}
	if v.Hostname != "vc.local" {
		t.Errorf("got %+v", v)
	}
}

func TestGetVMware_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetVMware(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting VMware") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetVMware_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetVMware(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateVMware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vmware.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "hostname": "h"}, nil
	})
	c, _ := ts.NewClient(ctx)
	v, err := c.CreateVMware(ctx, &types.VMwareCreateRequest{Hostname: "h"})
	if err != nil {
		t.Fatalf("CreateVMware: %v", err)
	}
	if v.ID != 2 {
		t.Errorf("got %+v", v)
	}
}

func TestCreateVMware_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateVMware(ctx, &types.VMwareCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating VMware") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateVMware_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateVMware(ctx, &types.VMwareCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateVMware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vmware.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "hostname": "h2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	hn := "h2"
	v, err := c.UpdateVMware(ctx, 2, &types.VMwareUpdateRequest{Hostname: &hn})
	if err != nil {
		t.Fatalf("UpdateVMware: %v", err)
	}
	if v.Hostname != "h2" {
		t.Errorf("got %+v", v)
	}
}

func TestUpdateVMware_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateVMware(ctx, 2, &types.VMwareUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating VMware") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateVMware_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateVMware(ctx, 2, &types.VMwareUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteVMware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "vmware.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteVMware(ctx, 2); err != nil {
		t.Fatalf("DeleteVMware: %v", err)
	}
	if !saw {
		t.Error("server did not see vmware.delete")
	}
}

func TestDeleteVMware_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteVMware(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting VMware") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
