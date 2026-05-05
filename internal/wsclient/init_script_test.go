package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetInitScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "initshutdownscript.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 3, "type": "COMMAND", "command": "echo hi",
			"when": "POSTINIT", "enabled": true, "timeout": 30,
		}, nil
	})
	c, _ := ts.NewClient(ctx)

	s, err := c.GetInitScript(ctx, 3)
	if err != nil {
		t.Fatalf("GetInitScript: %v", err)
	}
	if s.ID != 3 || s.Command != "echo hi" {
		t.Errorf("got %+v", s)
	}
}

func TestGetInitScript_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetInitScript(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting init script") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetInitScript_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetInitScript(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateInitScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "initshutdownscript.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 4, "type": "SCRIPT", "script": "/root/x.sh",
			"when": "PREINIT", "enabled": true,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.CreateInitScript(ctx, &types.InitScriptCreateRequest{
		Type: "SCRIPT", Script: "/root/x.sh", When: "PREINIT", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateInitScript: %v", err)
	}
	if s.ID != 4 {
		t.Errorf("ID: got %d", s.ID)
	}
}

func TestCreateInitScript_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInitScript(ctx, &types.InitScriptCreateRequest{Type: "SCRIPT", When: "PREINIT"})
	if err == nil || !strings.Contains(err.Error(), "creating init script") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateInitScript_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInitScript(ctx, &types.InitScriptCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateInitScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "initshutdownscript.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 4, "command": "updated"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.UpdateInitScript(ctx, 4, &types.InitScriptUpdateRequest{Command: "updated"})
	if err != nil {
		t.Fatalf("UpdateInitScript: %v", err)
	}
	if s.Command != "updated" {
		t.Errorf("got %q", s.Command)
	}
}

func TestUpdateInitScript_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateInitScript(ctx, 4, &types.InitScriptUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating init script") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateInitScript_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateInitScript(ctx, 4, &types.InitScriptUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteInitScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawDelete bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "initshutdownscript.delete" {
			sawDelete = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteInitScript(ctx, 4); err != nil {
		t.Fatalf("DeleteInitScript: %v", err)
	}
	if !sawDelete {
		t.Error("server did not see initshutdownscript.delete")
	}
}

func TestDeleteInitScript_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteInitScript(ctx, 4)
	if err == nil || !strings.Contains(err.Error(), "deleting init script") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
