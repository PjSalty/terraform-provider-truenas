package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetDirectoryServicesConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "directoryservices.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		st := "ACTIVEDIRECTORY"
		return map[string]interface{}{
			"id": 1, "service_type": st, "enable": true,
			"enable_account_cache": true, "timeout": 30,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetDirectoryServicesConfig(ctx)
	if err != nil {
		t.Fatalf("GetDirectoryServicesConfig: %v", err)
	}
	if cfg.ServiceType == nil || *cfg.ServiceType != "ACTIVEDIRECTORY" || !cfg.Enable {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetDirectoryServicesConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetDirectoryServicesConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting directory services") {
		t.Errorf("got %v", err)
	}
}

func TestGetDirectoryServicesConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetDirectoryServicesConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateDirectoryServicesConfig_directShape(t *testing.T) {
	// Server returns the full config object directly.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "directoryservices.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "enable": false, "timeout": 30}, nil
	})
	c, _ := ts.NewClient(ctx)
	enable := false
	cfg, err := c.UpdateDirectoryServicesConfig(ctx, &types.DirectoryServicesUpdateRequest{Enable: &enable})
	if err != nil {
		t.Fatalf("UpdateDirectoryServicesConfig: %v", err)
	}
	if cfg.Enable {
		t.Errorf("got %+v", cfg)
	}
}

func TestUpdateDirectoryServicesConfig_refetchFallback(t *testing.T) {
	// Server returns a non-config shape (e.g. job ID); client refetches.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "directoryservices.update":
			return float64(99), nil // job-id-as-int response
		case "directoryservices.config":
			return map[string]interface{}{"id": 1, "enable": true, "timeout": 60}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.UpdateDirectoryServicesConfig(ctx, &types.DirectoryServicesUpdateRequest{})
	if err != nil {
		t.Fatalf("UpdateDirectoryServicesConfig: %v", err)
	}
	if !cfg.Enable || cfg.Timeout != 60 {
		t.Errorf("got %+v", cfg)
	}
}

func TestUpdateDirectoryServicesConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateDirectoryServicesConfig(ctx, &types.DirectoryServicesUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating directory services") {
		t.Errorf("got %v", err)
	}
}

func TestLeaveDirectoryServices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "directoryservices.leave" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	raw, err := c.LeaveDirectoryServices(ctx, map[string]interface{}{"username": "admin"})
	if err != nil {
		t.Fatalf("LeaveDirectoryServices: %v", err)
	}
	if string(raw) != "true" {
		t.Errorf("got %q", string(raw))
	}
}

func TestLeaveDirectoryServices_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.LeaveDirectoryServices(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "leaving directory services") {
		t.Errorf("got %v", err)
	}
}
