package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetNetworkConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "network.configuration.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":          1,
			"hostname":    "tn",
			"nameserver1": "1.1.1.1",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetNetworkConfig(ctx)
	if err != nil {
		t.Fatalf("GetNetworkConfig: %v", err)
	}
	if cfg.Nameserver1 != "1.1.1.1" {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetNetworkConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNetworkConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting network configuration") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNetworkConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNetworkConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNetworkConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "network.configuration.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "nameserver1": "8.8.8.8"}, nil
	})
	c, _ := ts.NewClient(ctx)
	ns := "8.8.8.8"
	cfg, err := c.UpdateNetworkConfig(ctx, &types.NetworkConfigUpdateRequest{Nameserver1: &ns})
	if err != nil {
		t.Fatalf("UpdateNetworkConfig: %v", err)
	}
	if cfg.Nameserver1 != "8.8.8.8" {
		t.Errorf("got %+v", cfg)
	}
}

func TestUpdateNetworkConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNetworkConfig(ctx, &types.NetworkConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating network configuration") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNetworkConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNetworkConfig(ctx, &types.NetworkConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetFullNetworkConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "network.configuration.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":       1,
			"hostname": "tn",
			"hosts":    []interface{}{"127.0.0.1 localhost"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetFullNetworkConfig(ctx)
	if err != nil {
		t.Fatalf("GetFullNetworkConfig: %v", err)
	}
	if cfg.Hostname != "tn" || len(cfg.Hosts) != 1 {
		t.Errorf("got %+v", cfg)
	}
}

func TestGetFullNetworkConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFullNetworkConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting full network configuration") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetFullNetworkConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFullNetworkConfig(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateFullNetworkConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "network.configuration.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "hostname": "newhost"}, nil
	})
	c, _ := ts.NewClient(ctx)
	hostname := "newhost"
	cfg, err := c.UpdateFullNetworkConfig(ctx, &types.FullNetworkConfigUpdateRequest{Hostname: &hostname})
	if err != nil {
		t.Fatalf("UpdateFullNetworkConfig: %v", err)
	}
	if cfg.Hostname != "newhost" {
		t.Errorf("got %+v", cfg)
	}
}

func TestUpdateFullNetworkConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateFullNetworkConfig(ctx, &types.FullNetworkConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating full network configuration") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateFullNetworkConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateFullNetworkConfig(ctx, &types.FullNetworkConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}
