package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetTunable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "tunable.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 7, "type": "SYSCTL", "var": "net.ipv4.ip_forward",
			"value": "1", "comment": "router", "enabled": true,
		}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tun, err := c.GetTunable(ctx, 7)
	if err != nil {
		t.Fatalf("GetTunable: %v", err)
	}
	if tun.ID != 7 || tun.Var != "net.ipv4.ip_forward" || tun.Value != "1" {
		t.Errorf("unexpected tunable: %+v", tun)
	}
}

func TestGetTunable_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetTunable(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "tunable") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetTunable_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetTunable(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateTunable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "tunable.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 9, "type": "SYSCTL", "var": "vm.swappiness",
			"value": "10", "enabled": true,
		}, nil
	})
	c, _ := ts.NewClient(ctx)

	tun, err := c.CreateTunable(ctx, &types.TunableCreateRequest{
		Type: "SYSCTL", Var: "vm.swappiness", Value: "10", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateTunable: %v", err)
	}
	if tun.ID != 9 {
		t.Errorf("ID: got %d", tun.ID)
	}
}

func TestCreateTunable_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateTunable(ctx, &types.TunableCreateRequest{Type: "SYSCTL", Var: "x", Value: "1"})
	if err == nil || !strings.Contains(err.Error(), "creating tunable") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateTunable_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateTunable(ctx, &types.TunableCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestListTunables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []map[string]interface{}{
			{"id": 1, "var": "a", "value": "1"},
			{"id": 2, "var": "b", "value": "2"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)

	tunables, err := c.ListTunables(ctx)
	if err != nil {
		t.Fatalf("ListTunables: %v", err)
	}
	if len(tunables) != 2 {
		t.Errorf("got %d, want 2", len(tunables))
	}
}

func TestListTunables_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListTunables(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing") {
		t.Errorf("expected err, got %v", err)
	}
}

func TestListTunables_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListTunables(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestFindTunableByVar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []map[string]interface{}{
			{"id": 1, "var": "a", "value": "1"},
			{"id": 2, "var": "b", "value": "2"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)

	tun, err := c.FindTunableByVar(ctx, "b")
	if err != nil {
		t.Fatalf("FindTunableByVar: %v", err)
	}
	if tun.ID != 2 {
		t.Errorf("ID: got %d, want 2", tun.ID)
	}
}

func TestFindTunableByVar_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []map[string]interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.FindTunableByVar(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found err, got %v", err)
	}
}

func TestFindTunableByVar_listError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.FindTunableByVar(ctx, "x")
	if err == nil {
		t.Error("expected list err to bubble")
	}
}

func TestUpdateTunable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "tunable.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 7, "value": "2"}, nil
	})
	c, _ := ts.NewClient(ctx)

	tun, err := c.UpdateTunable(ctx, 7, &types.TunableUpdateRequest{Value: "2"})
	if err != nil {
		t.Fatalf("UpdateTunable: %v", err)
	}
	if tun.ID != 7 || tun.Value != "2" {
		t.Errorf("got %+v", tun)
	}
}

func TestUpdateTunable_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateTunable(ctx, 7, &types.TunableUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating tunable") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateTunable_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateTunable(ctx, 7, &types.TunableUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteTunable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawDelete bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "tunable.delete" {
			sawDelete = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteTunable(ctx, 7); err != nil {
		t.Fatalf("DeleteTunable: %v", err)
	}
	if !sawDelete {
		t.Error("server did not see tunable.delete")
	}
}

func TestDeleteTunable_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteTunable(ctx, 7)
	if err == nil || !strings.Contains(err.Error(), "deleting tunable") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
