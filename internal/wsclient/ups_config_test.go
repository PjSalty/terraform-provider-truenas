package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetUPSConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "ups.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "mode": "MASTER", "identifier": "ups"}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetUPSConfig(ctx)
	if err != nil {
		t.Fatalf("GetUPSConfig: %v", err)
	}
	if cfg.Mode != "MASTER" {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetUPSConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetUPSConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting UPS config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetUPSConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetUPSConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateUPSConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "ups.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "mode": "SLAVE"}, nil
	})
	c, _ := ts.NewClient(ctx)
	mode := "SLAVE"
	cfg, err := c.UpdateUPSConfig(ctx, &types.UPSConfigUpdateRequest{Mode: &mode})
	if err != nil {
		t.Fatalf("UpdateUPSConfig: %v", err)
	}
	if cfg.Mode != "SLAVE" {
		t.Errorf("got %q", cfg.Mode)
	}
}

func TestUpdateUPSConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateUPSConfig(ctx, &types.UPSConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating UPS config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateUPSConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateUPSConfig(ctx, &types.UPSConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
