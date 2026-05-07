package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListServices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{"id": 1, "service": "smb", "enable": true, "state": "RUNNING"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	svcs, err := c.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(svcs) != 1 || svcs[0].Service != "smb" {
		t.Errorf("got %+v", svcs)
	}
}

func TestListServices_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListServices(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing services") {
		t.Errorf("got %v", err)
	}
}

func TestListServices_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListServices(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "service": "ssh", "enable": true}, nil
	})
	c, _ := ts.NewClient(ctx)
	svc, err := c.GetService(ctx, 5)
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if svc.Service != "ssh" {
		t.Errorf("got %+v", svc)
	}
}

func TestGetService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetService(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting service") {
		t.Errorf("got %v", err)
	}
}

func TestGetService_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetService(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetServiceByName_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{map[string]interface{}{"id": 7, "service": "smb"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	svc, err := c.GetServiceByName(ctx, "smb")
	if err != nil {
		t.Fatalf("GetServiceByName: %v", err)
	}
	if svc.ID != 7 {
		t.Errorf("got %+v", svc)
	}
}

func TestGetServiceByName_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetServiceByName(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("got %v", err)
	}
}

func TestGetServiceByName_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetServiceByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "listing services") {
		t.Errorf("got %v", err)
	}
}

func TestGetServiceByName_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetServiceByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.UpdateService(ctx, 5, &types.ServiceUpdateRequest{Enable: true}); err != nil {
		t.Errorf("UpdateService: %v", err)
	}
}

func TestUpdateService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.UpdateService(ctx, 5, &types.ServiceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating service") {
		t.Errorf("got %v", err)
	}
}

func TestStartService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.start" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StartService(ctx, "smb"); err != nil {
		t.Errorf("StartService: %v", err)
	}
}

func TestStartService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StartService(ctx, "smb")
	if err == nil || !strings.Contains(err.Error(), "starting service") {
		t.Errorf("got %v", err)
	}
}

func TestStopService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.stop" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StopService(ctx, "smb"); err != nil {
		t.Errorf("StopService: %v", err)
	}
}

func TestStopService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StopService(ctx, "smb")
	if err == nil || !strings.Contains(err.Error(), "stopping service") {
		t.Errorf("got %v", err)
	}
}
