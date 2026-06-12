package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// =============================================================================
// nvmet.global
// =============================================================================

func TestGetNVMetGlobal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.global.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "basenqn": "nqn.x"}, nil
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.GetNVMetGlobal(ctx)
	if err != nil {
		t.Fatalf("GetNVMetGlobal: %v", err)
	}
	if g.Basenqn != "nqn.x" {
		t.Errorf("got %+v", g)
	}
}

func TestGetNVMetGlobal_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetGlobal(ctx)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet global") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetGlobal_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetGlobal(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNVMetGlobal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.global.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "basenqn": "nqn.y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	bn := "nqn.y"
	g, err := c.UpdateNVMetGlobal(ctx, &types.NVMetGlobalUpdateRequest{Basenqn: &bn})
	if err != nil {
		t.Fatalf("UpdateNVMetGlobal: %v", err)
	}
	if g.Basenqn != "nqn.y" {
		t.Errorf("got %+v", g)
	}
}

func TestUpdateNVMetGlobal_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetGlobal(ctx, &types.NVMetGlobalUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating nvmet global") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNVMetGlobal_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetGlobal(ctx, &types.NVMetGlobalUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

// =============================================================================
// nvmet.host
// =============================================================================

func TestGetNVMetHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.host.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "hostnqn": "nqn.host"}, nil
	})
	c, _ := ts.NewClient(ctx)
	h, err := c.GetNVMetHost(ctx, 1)
	if err != nil {
		t.Fatalf("GetNVMetHost: %v", err)
	}
	if h.Hostnqn != "nqn.host" {
		t.Errorf("got %+v", h)
	}
}

func TestGetNVMetHost_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetHost(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet host") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetHost_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetHost(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNVMetHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.host.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "hostnqn": "nqn.new"}, nil
	})
	c, _ := ts.NewClient(ctx)
	h, err := c.CreateNVMetHost(ctx, &types.NVMetHostCreateRequest{Hostnqn: "nqn.new"})
	if err != nil {
		t.Fatalf("CreateNVMetHost: %v", err)
	}
	if h.ID != 2 {
		t.Errorf("got %+v", h)
	}
}

func TestCreateNVMetHost_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetHost(ctx, &types.NVMetHostCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating nvmet host") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNVMetHost_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetHost(ctx, &types.NVMetHostCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNVMetHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.host.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "hostnqn": "nqn.up"}, nil
	})
	c, _ := ts.NewClient(ctx)
	hn := "nqn.up"
	h, err := c.UpdateNVMetHost(ctx, 2, &types.NVMetHostUpdateRequest{Hostnqn: &hn})
	if err != nil {
		t.Fatalf("UpdateNVMetHost: %v", err)
	}
	if h.Hostnqn != "nqn.up" {
		t.Errorf("got %+v", h)
	}
}

func TestUpdateNVMetHost_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetHost(ctx, 2, &types.NVMetHostUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating nvmet host") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNVMetHost_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetHost(ctx, 2, &types.NVMetHostUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNVMetHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "nvmet.host.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNVMetHost(ctx, 2); err != nil {
		t.Fatalf("DeleteNVMetHost: %v", err)
	}
	if !saw {
		t.Error("server did not see nvmet.host.delete")
	}
}

func TestDeleteNVMetHost_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNVMetHost(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting nvmet host") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

// =============================================================================
// nvmet.subsys
// =============================================================================

func TestGetNVMetSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.subsys.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "s"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.GetNVMetSubsys(ctx, 1)
	if err != nil {
		t.Fatalf("GetNVMetSubsys: %v", err)
	}
	if s.Name != "s" {
		t.Errorf("got %+v", s)
	}
}

func TestGetNVMetSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetSubsys(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetSubsys(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNVMetSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.subsys.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "ns"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.CreateNVMetSubsys(ctx, &types.NVMetSubsysCreateRequest{Name: "ns"})
	if err != nil {
		t.Fatalf("CreateNVMetSubsys: %v", err)
	}
	if s.ID != 2 {
		t.Errorf("got %+v", s)
	}
}

func TestCreateNVMetSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetSubsys(ctx, &types.NVMetSubsysCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating nvmet subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNVMetSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetSubsys(ctx, &types.NVMetSubsysCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNVMetSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.subsys.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	name := "renamed"
	s, err := c.UpdateNVMetSubsys(ctx, 2, &types.NVMetSubsysUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateNVMetSubsys: %v", err)
	}
	if s.Name != "renamed" {
		t.Errorf("got %+v", s)
	}
}

func TestUpdateNVMetSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetSubsys(ctx, 2, &types.NVMetSubsysUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating nvmet subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNVMetSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetSubsys(ctx, 2, &types.NVMetSubsysUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNVMetSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "nvmet.subsys.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNVMetSubsys(ctx, 2); err != nil {
		t.Fatalf("DeleteNVMetSubsys: %v", err)
	}
	if !saw {
		t.Error("server did not see nvmet.subsys.delete")
	}
}

func TestDeleteNVMetSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNVMetSubsys(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting nvmet subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

// =============================================================================
// nvmet.port
// =============================================================================

func TestGetNVMetPort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.port.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "addr_trtype": "TCP", "addr_traddr": "0.0.0.0"}, nil
	})
	c, _ := ts.NewClient(ctx)
	p, err := c.GetNVMetPort(ctx, 1)
	if err != nil {
		t.Fatalf("GetNVMetPort: %v", err)
	}
	if p.AddrTrtype != "TCP" {
		t.Errorf("got %+v", p)
	}
}

func TestGetNVMetPort_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetPort(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet port") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetPort_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetPort(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNVMetPort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.port.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "addr_trtype": "TCP"}, nil
	})
	c, _ := ts.NewClient(ctx)
	p, err := c.CreateNVMetPort(ctx, &types.NVMetPortCreateRequest{AddrTrtype: "TCP", AddrTraddr: "0.0.0.0"})
	if err != nil {
		t.Fatalf("CreateNVMetPort: %v", err)
	}
	if p.ID != 2 {
		t.Errorf("got %+v", p)
	}
}

func TestCreateNVMetPort_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetPort(ctx, &types.NVMetPortCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating nvmet port") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNVMetPort_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetPort(ctx, &types.NVMetPortCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNVMetPort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.port.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "addr_trtype": "RDMA"}, nil
	})
	c, _ := ts.NewClient(ctx)
	tt := "RDMA"
	p, err := c.UpdateNVMetPort(ctx, 2, &types.NVMetPortUpdateRequest{AddrTrtype: &tt})
	if err != nil {
		t.Fatalf("UpdateNVMetPort: %v", err)
	}
	if p.AddrTrtype != "RDMA" {
		t.Errorf("got %+v", p)
	}
}

func TestUpdateNVMetPort_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetPort(ctx, 2, &types.NVMetPortUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating nvmet port") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNVMetPort_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetPort(ctx, 2, &types.NVMetPortUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNVMetPort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "nvmet.port.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNVMetPort(ctx, 2); err != nil {
		t.Fatalf("DeleteNVMetPort: %v", err)
	}
	if !saw {
		t.Error("server did not see nvmet.port.delete")
	}
}

func TestDeleteNVMetPort_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNVMetPort(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting nvmet port") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

// =============================================================================
// nvmet.namespace
// =============================================================================

func TestGetNVMetNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.namespace.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "device_type": "ZVOL", "device_path": "/dev/zvol/x"}, nil
	})
	c, _ := ts.NewClient(ctx)
	n, err := c.GetNVMetNamespace(ctx, 1)
	if err != nil {
		t.Fatalf("GetNVMetNamespace: %v", err)
	}
	if n.DeviceType != "ZVOL" {
		t.Errorf("got %+v", n)
	}
}

func TestGetNVMetNamespace_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetNamespace(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet namespace") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetNamespace_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetNamespace(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNVMetNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.namespace.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "device_type": "FILE"}, nil
	})
	c, _ := ts.NewClient(ctx)
	n, err := c.CreateNVMetNamespace(ctx, &types.NVMetNamespaceCreateRequest{DeviceType: "FILE", DevicePath: "/x", SubsysID: 1})
	if err != nil {
		t.Fatalf("CreateNVMetNamespace: %v", err)
	}
	if n.ID != 2 {
		t.Errorf("got %+v", n)
	}
}

func TestCreateNVMetNamespace_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetNamespace(ctx, &types.NVMetNamespaceCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating nvmet namespace") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNVMetNamespace_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetNamespace(ctx, &types.NVMetNamespaceCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNVMetNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.namespace.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "device_path": "/y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	dp := "/y"
	n, err := c.UpdateNVMetNamespace(ctx, 2, &types.NVMetNamespaceUpdateRequest{DevicePath: &dp})
	if err != nil {
		t.Fatalf("UpdateNVMetNamespace: %v", err)
	}
	if n.DevicePath != "/y" {
		t.Errorf("got %+v", n)
	}
}

func TestUpdateNVMetNamespace_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetNamespace(ctx, 2, &types.NVMetNamespaceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating nvmet namespace") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNVMetNamespace_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNVMetNamespace(ctx, 2, &types.NVMetNamespaceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNVMetNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "nvmet.namespace.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNVMetNamespace(ctx, 2); err != nil {
		t.Fatalf("DeleteNVMetNamespace: %v", err)
	}
	if !saw {
		t.Error("server did not see nvmet.namespace.delete")
	}
}

func TestDeleteNVMetNamespace_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNVMetNamespace(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting nvmet namespace") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

// =============================================================================
// nvmet.host_subsys
// =============================================================================

func TestGetNVMetHostSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.host_subsys.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "host_id": 5, "subsys_id": 3}, nil
	})
	c, _ := ts.NewClient(ctx)
	hs, err := c.GetNVMetHostSubsys(ctx, 1)
	if err != nil {
		t.Fatalf("GetNVMetHostSubsys: %v", err)
	}
	if hs.HostID != 5 {
		t.Errorf("got %+v", hs)
	}
}

func TestGetNVMetHostSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetHostSubsys(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet host_subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetHostSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetHostSubsys(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNVMetHostSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.host_subsys.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	hs, err := c.CreateNVMetHostSubsys(ctx, &types.NVMetHostSubsysCreateRequest{HostID: 1, SubsysID: 2})
	if err != nil {
		t.Fatalf("CreateNVMetHostSubsys: %v", err)
	}
	if hs.ID != 2 {
		t.Errorf("got %+v", hs)
	}
}

func TestCreateNVMetHostSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetHostSubsys(ctx, &types.NVMetHostSubsysCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating nvmet host_subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNVMetHostSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetHostSubsys(ctx, &types.NVMetHostSubsysCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNVMetHostSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "nvmet.host_subsys.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNVMetHostSubsys(ctx, 2); err != nil {
		t.Fatalf("DeleteNVMetHostSubsys: %v", err)
	}
	if !saw {
		t.Error("server did not see nvmet.host_subsys.delete")
	}
}

func TestDeleteNVMetHostSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNVMetHostSubsys(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting nvmet host_subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

// =============================================================================
// nvmet.port_subsys
// =============================================================================

func TestGetNVMetPortSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.port_subsys.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "port_id": 5, "subsys_id": 3}, nil
	})
	c, _ := ts.NewClient(ctx)
	ps, err := c.GetNVMetPortSubsys(ctx, 1)
	if err != nil {
		t.Fatalf("GetNVMetPortSubsys: %v", err)
	}
	if ps.PortID != 5 {
		t.Errorf("got %+v", ps)
	}
}

func TestGetNVMetPortSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetPortSubsys(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting nvmet port_subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNVMetPortSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNVMetPortSubsys(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNVMetPortSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "nvmet.port_subsys.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	ps, err := c.CreateNVMetPortSubsys(ctx, &types.NVMetPortSubsysCreateRequest{PortID: 1, SubsysID: 2})
	if err != nil {
		t.Fatalf("CreateNVMetPortSubsys: %v", err)
	}
	if ps.ID != 2 {
		t.Errorf("got %+v", ps)
	}
}

func TestCreateNVMetPortSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetPortSubsys(ctx, &types.NVMetPortSubsysCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating nvmet port_subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNVMetPortSubsys_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNVMetPortSubsys(ctx, &types.NVMetPortSubsysCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNVMetPortSubsys(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "nvmet.port_subsys.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNVMetPortSubsys(ctx, 2); err != nil {
		t.Fatalf("DeleteNVMetPortSubsys: %v", err)
	}
	if !saw {
		t.Error("server did not see nvmet.port_subsys.delete")
	}
}

func TestDeleteNVMetPortSubsys_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNVMetPortSubsys(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting nvmet port_subsys") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
