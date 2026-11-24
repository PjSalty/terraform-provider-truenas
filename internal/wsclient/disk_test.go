package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListDisks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "disk.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{
				"identifier": "{serial}WD-X1", "name": "sda", "serial": "WD-X1",
				"size": 4000787030016, "model": "WDC WD40", "type": "HDD",
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	disks, err := c.ListDisks(ctx)
	if err != nil {
		t.Fatalf("ListDisks: %v", err)
	}
	if len(disks) != 1 || disks[0].Name != "sda" {
		t.Errorf("got %+v", disks)
	}
}

func TestListDisks_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListDisks(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing disks") {
		t.Errorf("got %v", err)
	}
}

func TestListDisks_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListDisks(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetDisk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "disk.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"identifier": "{serial}WD-X1", "name": "sda",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.GetDisk(ctx, "{serial}WD-X1")
	if err != nil {
		t.Fatalf("GetDisk: %v", err)
	}
	if d.Name != "sda" {
		t.Errorf("got %+v", d)
	}
}

func TestGetDisk_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetDisk(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "getting disk") {
		t.Errorf("got %v", err)
	}
}

func TestGetDisk_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetDisk(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// disk.query reports every pool as null unless extra.pools is set, so
// GetDisk has to ask for it, and it has to decode the two power settings.
func TestGetDisk_asksForPoolsAndDecodesSettings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var params []interface{}
	ts := NewTestServer(t, func(_ context.Context, _ string, p []interface{}) (interface{}, *RPCError) {
		params = p
		return map[string]interface{}{
			"identifier": "{serial}WD-X1", "name": "sda", "pool": "tank",
			"hddstandby": "ALWAYS ON", "advpowermgmt": "128",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	d, err := c.GetDisk(ctx, "{serial}WD-X1")
	if err != nil {
		t.Fatalf("GetDisk: %v", err)
	}
	if d.HDDStandby != "ALWAYS ON" || d.AdvPowerMgmt != "128" || d.Pool == nil || *d.Pool != "tank" {
		t.Errorf("got %+v", d)
	}
	if len(params) != 2 || params[0] != "{serial}WD-X1" {
		t.Fatalf("params = %#v", params)
	}
	opts, _ := params[1].(map[string]interface{})
	extra, _ := opts["extra"].(map[string]interface{})
	if extra["pools"] != true {
		t.Errorf("options = %#v, want extra.pools", params[1])
	}
}

func TestUpdateDisk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var params []interface{}
	ts := NewTestServer(t, func(_ context.Context, m string, p []interface{}) (interface{}, *RPCError) {
		if m != "disk.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		}
		params = p
		return map[string]interface{}{"identifier": "{serial}WD-X1"}, nil
	})
	c, _ := ts.NewClient(ctx)

	// Only what is set goes on the wire: disk.update is a partial update.
	standby := "60"
	if err := c.UpdateDisk(ctx, "{serial}WD-X1", &types.DiskUpdateRequest{HDDStandby: &standby}); err != nil {
		t.Fatalf("UpdateDisk: %v", err)
	}
	if len(params) != 2 || params[0] != "{serial}WD-X1" {
		t.Fatalf("params = %#v", params)
	}
	body, _ := params[1].(map[string]interface{})
	if len(body) != 1 || body["hddstandby"] != "60" {
		t.Errorf("body = %#v, want only hddstandby", body)
	}

	// An empty string is a real description, not an unset one.
	empty := ""
	if err := c.UpdateDisk(ctx, "x", &types.DiskUpdateRequest{Description: &empty}); err != nil {
		t.Fatalf("UpdateDisk: %v", err)
	}
	body, _ = params[1].(map[string]interface{})
	if v, ok := body["description"]; !ok || v != "" {
		t.Errorf("body = %#v, want an empty description", body)
	}
}

func TestUpdateDisk_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EINVAL] nope"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.UpdateDisk(ctx, "x", &types.DiskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating disk") {
		t.Errorf("got %v", err)
	}
}
