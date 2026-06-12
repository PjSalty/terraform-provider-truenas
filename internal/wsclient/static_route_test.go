package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetStaticRoute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "staticroute.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":          11,
			"destination": "10.0.0.0/8",
			"gateway":     "192.168.1.1",
			"description": "lan",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetStaticRoute(ctx, 11)
	if err != nil {
		t.Fatalf("GetStaticRoute: %v", err)
	}
	if r.Destination != "10.0.0.0/8" {
		t.Errorf("got %+v", r)
	}
}

func TestGetStaticRoute_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetStaticRoute(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting static route") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetStaticRoute_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetStaticRoute(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateStaticRoute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "staticroute.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":          12,
			"destination": "172.16.0.0/12",
			"gateway":     "192.168.1.1",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateStaticRoute(ctx, &types.StaticRouteCreateRequest{
		Destination: "172.16.0.0/12",
		Gateway:     "192.168.1.1",
	})
	if err != nil {
		t.Fatalf("CreateStaticRoute: %v", err)
	}
	if r.ID != 12 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateStaticRoute_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateStaticRoute(ctx, &types.StaticRouteCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating static route") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateStaticRoute_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateStaticRoute(ctx, &types.StaticRouteCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateStaticRoute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "staticroute.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 12, "description": "updated"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateStaticRoute(ctx, 12, &types.StaticRouteUpdateRequest{Description: "updated"})
	if err != nil {
		t.Fatalf("UpdateStaticRoute: %v", err)
	}
	if r.Description != "updated" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateStaticRoute_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateStaticRoute(ctx, 12, &types.StaticRouteUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating static route") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateStaticRoute_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateStaticRoute(ctx, 12, &types.StaticRouteUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteStaticRoute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "staticroute.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteStaticRoute(ctx, 12); err != nil {
		t.Fatalf("DeleteStaticRoute: %v", err)
	}
	if !saw {
		t.Error("server did not see staticroute.delete")
	}
}

func TestDeleteStaticRoute_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteStaticRoute(ctx, 12)
	if err == nil || !strings.Contains(err.Error(), "deleting static route") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
