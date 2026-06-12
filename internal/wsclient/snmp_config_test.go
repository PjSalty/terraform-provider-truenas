package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetSNMPConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "snmp.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "community": "public"}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetSNMPConfig(ctx)
	if err != nil {
		t.Fatalf("GetSNMPConfig: %v", err)
	}
	if cfg.Community != "public" {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetSNMPConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSNMPConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting SNMP config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetSNMPConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSNMPConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateSNMPConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "snmp.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "community": "private"}, nil
	})
	c, _ := ts.NewClient(ctx)
	community := "private"
	cfg, err := c.UpdateSNMPConfig(ctx, &types.SNMPConfigUpdateRequest{Community: &community})
	if err != nil {
		t.Fatalf("UpdateSNMPConfig: %v", err)
	}
	if cfg.Community != "private" {
		t.Errorf("got %q", cfg.Community)
	}
}

func TestUpdateSNMPConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSNMPConfig(ctx, &types.SNMPConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating SNMP config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateSNMPConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSNMPConfig(ctx, &types.SNMPConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
