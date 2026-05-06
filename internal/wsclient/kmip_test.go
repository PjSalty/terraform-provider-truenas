package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetKMIPConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kmip.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "enabled": false, "port": 5696}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetKMIPConfig(ctx)
	if err != nil {
		t.Fatalf("GetKMIPConfig: %v", err)
	}
	if cfg.Port != 5696 {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetKMIPConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKMIPConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting KMIP config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetKMIPConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKMIPConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateKMIPConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kmip.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "enabled": true}, nil
	})
	c, _ := ts.NewClient(ctx)
	enabled := true
	cfg, err := c.UpdateKMIPConfig(ctx, &types.KMIPUpdateRequest{Enabled: &enabled})
	if err != nil {
		t.Fatalf("UpdateKMIPConfig: %v", err)
	}
	if !cfg.Enabled {
		t.Errorf("got %+v", cfg)
	}
}

// TestUpdateKMIPConfig_jobResponseFallback covers the branch where the
// update response decodes but with ID=0, triggering a re-fetch via
// GetKMIPConfig.
func TestUpdateKMIPConfig_jobResponseFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "kmip.update":
			// Response has no ID, simulating a job-style response.
			return map[string]interface{}{}, nil
		case "kmip.config":
			return map[string]interface{}{"id": 1, "enabled": false, "port": 5696}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.UpdateKMIPConfig(ctx, &types.KMIPUpdateRequest{})
	if err != nil {
		t.Fatalf("UpdateKMIPConfig: %v", err)
	}
	if cfg.ID != 1 {
		t.Errorf("expected refetched config, got %+v", cfg)
	}
}

func TestUpdateKMIPConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKMIPConfig(ctx, &types.KMIPUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating KMIP config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
