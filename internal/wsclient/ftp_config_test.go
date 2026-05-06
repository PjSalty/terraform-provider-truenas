package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetFTPConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "ftp.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":   1,
			"port": 21,
			"tls":  true,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetFTPConfig(ctx)
	if err != nil {
		t.Fatalf("GetFTPConfig: %v", err)
	}
	if cfg.Port != 21 || !cfg.TLS {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetFTPConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFTPConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting FTP config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetFTPConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFTPConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateFTPConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "ftp.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "port": 2121}, nil
	})
	c, _ := ts.NewClient(ctx)
	port := 2121
	cfg, err := c.UpdateFTPConfig(ctx, &types.FTPConfigUpdateRequest{Port: &port})
	if err != nil {
		t.Fatalf("UpdateFTPConfig: %v", err)
	}
	if cfg.Port != 2121 {
		t.Errorf("got %d", cfg.Port)
	}
}

func TestUpdateFTPConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateFTPConfig(ctx, &types.FTPConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating FTP config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateFTPConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateFTPConfig(ctx, &types.FTPConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
