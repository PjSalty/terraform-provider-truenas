package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetSMBConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "smb.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":          1,
			"netbiosname": "truenas",
			"workgroup":   "WORKGROUP",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetSMBConfig(ctx)
	if err != nil {
		t.Fatalf("GetSMBConfig: %v", err)
	}
	if cfg.NetbiosName != "truenas" {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetSMBConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSMBConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting SMB config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetSMBConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSMBConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateSMBConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "smb.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "netbiosname": "newname"}, nil
	})
	c, _ := ts.NewClient(ctx)
	name := "newname"
	cfg, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{NetbiosName: &name})
	if err != nil {
		t.Fatalf("UpdateSMBConfig: %v", err)
	}
	if cfg.NetbiosName != "newname" {
		t.Errorf("got %q", cfg.NetbiosName)
	}
}

func TestUpdateSMBConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating SMB config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateSMBConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
