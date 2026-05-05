package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGetSystemInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "system.info" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"version":  "TrueNAS-SCALE-25.04.2",
			"hostname": "truenas",
			"physmem":  16777216000,
			"cores":    8,
		}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	info, err := c.GetSystemInfo(ctx)
	if err != nil {
		t.Fatalf("GetSystemInfo: %v", err)
	}
	if info.Version != "TrueNAS-SCALE-25.04.2" {
		t.Errorf("Version: got %q", info.Version)
	}
	if info.Hostname != "truenas" {
		t.Errorf("Hostname: got %q", info.Hostname)
	}
	if info.Cores != 8 {
		t.Errorf("Cores: got %d", info.Cores)
	}
}

func TestGetSystemInfo_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetSystemInfo(ctx)
	if err == nil || !strings.Contains(err.Error(), "system info") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestGetSystemInfo_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetSystemInfo(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse error, got %v", err)
	}
}
