package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetSSHConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "ssh.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "tcpport": 22}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetSSHConfig(ctx)
	if err != nil {
		t.Fatalf("GetSSHConfig: %v", err)
	}
	if cfg.TCPPort != 22 {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetSSHConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSSHConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting SSH config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetSSHConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSSHConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateSSHConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "ssh.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "tcpport": 2222}, nil
	})
	c, _ := ts.NewClient(ctx)
	port := 2222
	cfg, err := c.UpdateSSHConfig(ctx, &types.SSHConfigUpdateRequest{TCPPort: &port})
	if err != nil {
		t.Fatalf("UpdateSSHConfig: %v", err)
	}
	if cfg.TCPPort != 2222 {
		t.Errorf("got %d", cfg.TCPPort)
	}
}

func TestUpdateSSHConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSSHConfig(ctx, &types.SSHConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating SSH config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateSSHConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSSHConfig(ctx, &types.SSHConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
