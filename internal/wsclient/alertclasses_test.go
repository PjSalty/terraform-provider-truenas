package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetAlertClassesConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "alertclasses.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1,
			"classes": map[string]interface{}{
				"VolumeStatus": map[string]interface{}{
					"level":  "WARNING",
					"policy": "IMMEDIATELY",
				},
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)

	cfg, err := c.GetAlertClassesConfig(ctx)
	if err != nil {
		t.Fatalf("GetAlertClassesConfig: %v", err)
	}
	if cfg.Classes["VolumeStatus"].Level != "WARNING" {
		t.Errorf("got %+v", cfg.Classes)
	}
}

func TestGetAlertClassesConfig_nilClassesNormalized(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return map[string]interface{}{"id": 1}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetAlertClassesConfig(ctx)
	if err != nil {
		t.Fatalf("GetAlertClassesConfig: %v", err)
	}
	if cfg.Classes == nil {
		t.Error("Classes should be non-nil after normalization")
	}
}

func TestGetAlertClassesConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAlertClassesConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting alert classes config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetAlertClassesConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAlertClassesConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateAlertClassesConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "alertclasses.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1,
			"classes": map[string]interface{}{
				"PoolHealth": map[string]interface{}{
					"level":  "CRITICAL",
					"policy": "IMMEDIATELY",
				},
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.UpdateAlertClassesConfig(ctx, &types.AlertClassesUpdateRequest{
		Classes: map[string]types.AlertClassEntry{
			"PoolHealth": {Level: "CRITICAL", Policy: "IMMEDIATELY"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateAlertClassesConfig: %v", err)
	}
	if cfg.Classes["PoolHealth"].Level != "CRITICAL" {
		t.Errorf("got %+v", cfg.Classes)
	}
}

func TestUpdateAlertClassesConfig_nilClassesNormalized(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return map[string]interface{}{"id": 1}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.UpdateAlertClassesConfig(ctx, &types.AlertClassesUpdateRequest{
		Classes: map[string]types.AlertClassEntry{},
	})
	if err != nil {
		t.Fatalf("UpdateAlertClassesConfig: %v", err)
	}
	if cfg.Classes == nil {
		t.Error("Classes should be non-nil after normalization")
	}
}

func TestUpdateAlertClassesConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateAlertClassesConfig(ctx, &types.AlertClassesUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating alert classes config") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateAlertClassesConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateAlertClassesConfig(ctx, &types.AlertClassesUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
