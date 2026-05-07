package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestCreateZvol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawType string
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		if len(params) > 0 {
			if m, ok := params[0].(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok {
					sawType = t
				}
			}
		}
		return map[string]interface{}{"id": "tank/zv", "name": "tank/zv", "type": "VOLUME"}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.CreateZvol(ctx, &types.ZvolCreateRequest{Name: "tank/zv", Volsize: 1 << 24})
	if err != nil {
		t.Fatalf("CreateZvol: %v", err)
	}
	if d.ID != "tank/zv" {
		t.Errorf("got %+v", d)
	}
	if sawType != "VOLUME" {
		t.Errorf("server did not see Type=VOLUME on the request, saw %q", sawType)
	}
}

func TestCreateZvol_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateZvol(ctx, &types.ZvolCreateRequest{Name: "x", Volsize: 1})
	if err == nil || !strings.Contains(err.Error(), "creating zvol") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateZvol_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateZvol(ctx, &types.ZvolCreateRequest{Name: "x", Volsize: 1})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetZvol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": "tank/zv", "name": "tank/zv", "type": "VOLUME"}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.GetZvol(ctx, "tank/zv")
	if err != nil {
		t.Fatalf("GetZvol: %v", err)
	}
	if d.ID != "tank/zv" {
		t.Errorf("got %+v", d)
	}
}

func TestUpdateZvol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.dataset.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": "tank/zv", "name": "tank/zv"}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.UpdateZvol(ctx, "tank/zv", &types.ZvolUpdateRequest{Volsize: 1 << 25})
	if err != nil {
		t.Fatalf("UpdateZvol: %v", err)
	}
	if d.ID != "tank/zv" {
		t.Errorf("got %+v", d)
	}
}

func TestUpdateZvol_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateZvol(ctx, "tank/zv", &types.ZvolUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating zvol") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateZvol_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateZvol(ctx, "tank/zv", &types.ZvolUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteZvol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "pool.dataset.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteZvol(ctx, "tank/zv"); err != nil {
		t.Fatalf("DeleteZvol: %v", err)
	}
	if !saw {
		t.Error("server did not see pool.dataset.delete")
	}
}
