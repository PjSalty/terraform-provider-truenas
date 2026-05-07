package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListVMs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "name": "vm1"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	vs, err := c.ListVMs(ctx)
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if len(vs) != 1 {
		t.Errorf("got %+v", vs)
	}
}

func TestListVMs_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListVMs(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing VMs") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListVMs_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListVMs(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetVM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "vm1"}, nil
	})
	c, _ := ts.NewClient(ctx)
	v, err := c.GetVM(ctx, 1)
	if err != nil {
		t.Fatalf("GetVM: %v", err)
	}
	if v.Name != "vm1" {
		t.Errorf("got %+v", v)
	}
}

func TestGetVM_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetVM(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting VM") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetVM_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetVM(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateVM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "vm2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	v, err := c.CreateVM(ctx, &types.VMCreateRequest{Name: "vm2", Memory: 1024})
	if err != nil {
		t.Fatalf("CreateVM: %v", err)
	}
	if v.ID != 2 {
		t.Errorf("got %+v", v)
	}
}

func TestCreateVM_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateVM(ctx, &types.VMCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating VM") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateVM_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateVM(ctx, &types.VMCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateVM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	name := "renamed"
	v, err := c.UpdateVM(ctx, 2, &types.VMUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateVM: %v", err)
	}
	if v.Name != "renamed" {
		t.Errorf("got %+v", v)
	}
}

func TestUpdateVM_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateVM(ctx, 2, &types.VMUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating VM") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateVM_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateVM(ctx, 2, &types.VMUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteVM_default(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "vm.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteVM(ctx, 2, nil); err != nil {
		t.Fatalf("DeleteVM: %v", err)
	}
	if !saw {
		t.Error("server did not see vm.delete")
	}
}

func TestDeleteVM_withOpts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteVM(ctx, 2, &types.VMDeleteOptions{Force: false, Zvols: true}); err != nil {
		t.Fatalf("DeleteVM: %v", err)
	}
}

func TestDeleteVM_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteVM(ctx, 2, nil)
	if err == nil || !strings.Contains(err.Error(), "deleting VM") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestStartVM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "vm.start" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StartVM(ctx, 2); err != nil {
		t.Fatalf("StartVM: %v", err)
	}
	if !saw {
		t.Error("server did not see vm.start")
	}
}

func TestStartVM_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StartVM(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "starting VM") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestStopVM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "vm.stop" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StopVM(ctx, 2, true); err != nil {
		t.Fatalf("StopVM: %v", err)
	}
	if !saw {
		t.Error("server did not see vm.stop")
	}
}

func TestStopVM_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StopVM(ctx, 2, false)
	if err == nil || !strings.Contains(err.Error(), "stopping VM") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

// VM Device tests

func TestGetVMDevice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.device.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "vm": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.GetVMDevice(ctx, 1)
	if err != nil {
		t.Fatalf("GetVMDevice: %v", err)
	}
	if d.VM != 2 {
		t.Errorf("got %+v", d)
	}
}

func TestGetVMDevice_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetVMDevice(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting VM device") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetVMDevice_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetVMDevice(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateVMDevice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.device.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "vm": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.CreateVMDevice(ctx, &types.VMDeviceCreateRequest{VM: 2, Attributes: map[string]interface{}{"dtype": "DISK"}})
	if err != nil {
		t.Fatalf("CreateVMDevice: %v", err)
	}
	if d.ID != 5 {
		t.Errorf("got %+v", d)
	}
}

func TestCreateVMDevice_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateVMDevice(ctx, &types.VMDeviceCreateRequest{VM: 2})
	if err == nil || !strings.Contains(err.Error(), "creating VM device") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateVMDevice_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateVMDevice(ctx, &types.VMDeviceCreateRequest{VM: 2})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateVMDevice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "vm.device.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.UpdateVMDevice(ctx, 5, &types.VMDeviceUpdateRequest{})
	if err != nil {
		t.Fatalf("UpdateVMDevice: %v", err)
	}
	if d.ID != 5 {
		t.Errorf("got %+v", d)
	}
}

func TestUpdateVMDevice_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateVMDevice(ctx, 5, &types.VMDeviceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating VM device") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateVMDevice_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateVMDevice(ctx, 5, &types.VMDeviceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteVMDevice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "vm.device.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteVMDevice(ctx, 5); err != nil {
		t.Fatalf("DeleteVMDevice: %v", err)
	}
	if !saw {
		t.Error("server did not see vm.device.delete")
	}
}

func TestDeleteVMDevice_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteVMDevice(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "deleting VM device") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
