package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetNFSConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nfs.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":      1,
			"servers": 4,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetNFSConfig(ctx)
	if err != nil {
		t.Fatalf("GetNFSConfig: %v", err)
	}
	if cfg.Servers != 4 {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetNFSConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNFSConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting NFS config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNFSConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNFSConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNFSConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nfs.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "servers": 8}, nil
	})
	c, _ := ts.NewClient(ctx)
	srv := 8
	cfg, err := c.UpdateNFSConfig(ctx, &types.NFSConfigUpdateRequest{Servers: &srv})
	if err != nil {
		t.Fatalf("UpdateNFSConfig: %v", err)
	}
	if cfg.Servers != 8 {
		t.Errorf("got %d", cfg.Servers)
	}
}

func TestUpdateNFSConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNFSConfig(ctx, &types.NFSConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating NFS config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNFSConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNFSConfig(ctx, &types.NFSConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
