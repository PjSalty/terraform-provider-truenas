package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// commitHandler is a small helper that responds to interface.commit and
// interface.checkin so tests of the staged-commit flow can focus on the
// CRUD method under test.
func commitHandler(t *testing.T, methodOK string, payload interface{}) TestHandler {
	t.Helper()
	return func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.commit", "interface.checkin":
			return nil, nil
		case methodOK, "interface.get_instance":
			return payload, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	}
}

func TestGetInterface(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "interface.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": "br0", "name": "br0", "type": "BRIDGE"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetInterface(ctx, "br0")
	if err != nil {
		t.Fatalf("GetInterface: %v", err)
	}
	if r.Type != "BRIDGE" {
		t.Errorf("got %+v", r)
	}
}

func TestGetInterface_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetInterface(ctx, "br0")
	if err == nil || !strings.Contains(err.Error(), "getting interface") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetInterface_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetInterface(ctx, "br0")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestListInterfaces(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "interface.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": "br0"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	rs, err := c.ListInterfaces(ctx)
	if err != nil {
		t.Fatalf("ListInterfaces: %v", err)
	}
	if len(rs) != 1 {
		t.Errorf("got %+v", rs)
	}
}

func TestListInterfaces_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListInterfaces(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing interfaces") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListInterfaces_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListInterfaces(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateInterface(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, commitHandler(t, "interface.create",
		map[string]interface{}{"id": "br0", "type": "BRIDGE"}))
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateInterface(ctx, &types.NetworkInterfaceCreateRequest{Type: "BRIDGE"}, true)
	if err != nil {
		t.Fatalf("CreateInterface: %v", err)
	}
	if r.ID != "br0" {
		t.Errorf("got %+v", r)
	}
}

func TestCreateInterface_createError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "interface.create" {
			return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInterface(ctx, &types.NetworkInterfaceCreateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "creating interface") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateInterface_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "interface.create" {
			return "not-an-object", nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInterface(ctx, &types.NetworkInterfaceCreateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateInterface_commitError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.create":
			return map[string]interface{}{"id": "br0"}, nil
		case "interface.commit":
			return nil, &RPCError{Code: CodeInternalError, Message: "commit-fail"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInterface(ctx, &types.NetworkInterfaceCreateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "committing") {
		t.Errorf("expected commit err, got %v", err)
	}
}

func TestCreateInterface_checkinError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.create":
			return map[string]interface{}{"id": "br0"}, nil
		case "interface.commit":
			return nil, nil
		case "interface.checkin":
			return nil, &RPCError{Code: CodeInternalError, Message: "checkin-fail"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInterface(ctx, &types.NetworkInterfaceCreateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "checking in") {
		t.Errorf("expected checkin err, got %v", err)
	}
}

func TestUpdateInterface(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, commitHandler(t, "interface.update",
		map[string]interface{}{"id": "br0", "description": "u"}))
	c, _ := ts.NewClient(ctx)
	d := "u"
	r, err := c.UpdateInterface(ctx, "br0", &types.NetworkInterfaceUpdateRequest{Description: &d}, true)
	if err != nil {
		t.Fatalf("UpdateInterface: %v", err)
	}
	if r.ID != "br0" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateInterface_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "interface.update" {
			return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateInterface(ctx, "br0", &types.NetworkInterfaceUpdateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "updating interface") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateInterface_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "interface.update" {
			return "not-an-object", nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateInterface(ctx, "br0", &types.NetworkInterfaceUpdateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateInterface_commitError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.update":
			return map[string]interface{}{"id": "br0"}, nil
		case "interface.commit":
			return nil, &RPCError{Code: CodeInternalError, Message: "commit-fail"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateInterface(ctx, "br0", &types.NetworkInterfaceUpdateRequest{}, true)
	if err == nil || !strings.Contains(err.Error(), "committing") {
		t.Errorf("expected commit err, got %v", err)
	}
}

func TestDeleteInterface(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawDelete bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.delete":
			sawDelete = true
			return nil, nil
		case "interface.commit", "interface.checkin":
			return nil, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteInterface(ctx, "br0", true); err != nil {
		t.Fatalf("DeleteInterface: %v", err)
	}
	if !sawDelete {
		t.Error("server did not see interface.delete")
	}
}

func TestDeleteInterface_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "interface.delete" {
			return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteInterface(ctx, "br0", true)
	if err == nil || !strings.Contains(err.Error(), "deleting interface") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateInterface_noRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawCheckin bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.create":
			return map[string]interface{}{"id": "br0"}, nil
		case "interface.commit":
			return nil, nil
		case "interface.get_instance":
			return map[string]interface{}{"id": "br0", "type": "BRIDGE"}, nil
		case "interface.checkin":
			sawCheckin = true
			return nil, &RPCError{Code: CodeInternalError, Message: "should-not-be-called"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateInterface(ctx, &types.NetworkInterfaceCreateRequest{Type: "BRIDGE"}, false)
	if err != nil {
		t.Fatalf("CreateInterface with rollback=false: %v", err)
	}
	if sawCheckin {
		t.Error("interface.checkin should not be called when rollback=false")
	}
}

// TestUpdateInterface_commitPayload pins the interface.commit request body
// for both rollback modes: rollback passes through as sent and the checkin
// window is the 60-second middleware default. The window must stay longer
// than the client's idempotent-retry backoff (roughly 7.5s); a shorter one
// lets the middleware auto-revert a legitimate change mid-retry, which is
// exactly the regression this test guards against.
func TestUpdateInterface_commitPayload(t *testing.T) {
	for _, rollback := range []bool{true, false} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var commitBody map[string]interface{}
		var sawCheckin bool
		ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
			switch method {
			case "interface.update", "interface.get_instance":
				return map[string]interface{}{"id": "eth0", "type": "PHYSICAL"}, nil
			case "interface.commit":
				if len(params) == 1 {
					commitBody, _ = params[0].(map[string]interface{})
				}
				return nil, nil
			case "interface.checkin":
				sawCheckin = true
				return nil, nil
			}
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		})
		c, _ := ts.NewClient(ctx)
		if _, err := c.UpdateInterface(ctx, "eth0", &types.NetworkInterfaceUpdateRequest{}, rollback); err != nil {
			t.Fatalf("rollback=%v: UpdateInterface: %v", rollback, err)
		}
		if commitBody == nil {
			t.Fatalf("rollback=%v: interface.commit received no payload object", rollback)
		}
		if got, ok := commitBody["rollback"].(bool); !ok || got != rollback {
			t.Errorf("rollback=%v: commit payload rollback = %v", rollback, commitBody["rollback"])
		}
		if got, ok := commitBody["checkin_timeout"].(float64); !ok || got != 60 {
			t.Errorf("rollback=%v: commit payload checkin_timeout = %v, want 60", rollback, commitBody["checkin_timeout"])
		}
		if sawCheckin != rollback {
			t.Errorf("rollback=%v: sawCheckin=%v; checkin must run only when rollback is armed", rollback, sawCheckin)
		}
		cancel()
	}
}

func TestDeleteInterface_commitError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "interface.delete":
			return nil, nil
		case "interface.commit":
			return nil, &RPCError{Code: CodeInternalError, Message: "commit-fail"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteInterface(ctx, "br0", true)
	if err == nil || !strings.Contains(err.Error(), "committing") {
		t.Errorf("expected commit err, got %v", err)
	}
}
