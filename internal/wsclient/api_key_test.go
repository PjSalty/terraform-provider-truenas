package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetAPIKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "api_key.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "k", "username": "u"}, nil
	})
	c, _ := ts.NewClient(ctx)
	k, err := c.GetAPIKey(ctx, 1)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if k.Name != "k" {
		t.Errorf("got %+v", k)
	}
}

func TestGetAPIKey_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAPIKey(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting API key") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetAPIKey_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAPIKey(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateAPIKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "api_key.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "k2", "key": "secret"}, nil
	})
	c, _ := ts.NewClient(ctx)
	k, err := c.CreateAPIKey(ctx, &types.APIKeyCreateRequest{Name: "k2"})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if k.Key != "secret" {
		t.Errorf("got %+v", k)
	}
}

func TestCreateAPIKey_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateAPIKey(ctx, &types.APIKeyCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating API key") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateAPIKey_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateAPIKey(ctx, &types.APIKeyCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateAPIKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "api_key.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	k, err := c.UpdateAPIKey(ctx, 2, &types.APIKeyUpdateRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateAPIKey: %v", err)
	}
	if k.Name != "renamed" {
		t.Errorf("got %+v", k)
	}
}

func TestUpdateAPIKey_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateAPIKey(ctx, 2, &types.APIKeyUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating API key") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateAPIKey_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateAPIKey(ctx, 2, &types.APIKeyUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "api_key.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteAPIKey(ctx, 2); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	if !saw {
		t.Error("server did not see api_key.delete")
	}
}

func TestDeleteAPIKey_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteAPIKey(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting API key") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
