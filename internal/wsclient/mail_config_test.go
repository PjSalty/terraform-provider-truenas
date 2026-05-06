package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetMailConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "mail.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "fromemail": "ops@example.com", "port": 25}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetMailConfig(ctx)
	if err != nil {
		t.Fatalf("GetMailConfig: %v", err)
	}
	if cfg.FromEmail != "ops@example.com" {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetMailConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetMailConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting mail config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetMailConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetMailConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateMailConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "mail.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "outgoingserver": "smtp.example.com"}, nil
	})
	c, _ := ts.NewClient(ctx)
	server := "smtp.example.com"
	cfg, err := c.UpdateMailConfig(ctx, &types.MailConfigUpdateRequest{OutgoingServer: &server})
	if err != nil {
		t.Fatalf("UpdateMailConfig: %v", err)
	}
	if cfg.OutgoingServer != "smtp.example.com" {
		t.Errorf("got %q", cfg.OutgoingServer)
	}
}

func TestUpdateMailConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateMailConfig(ctx, &types.MailConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating mail config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateMailConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateMailConfig(ctx, &types.MailConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
