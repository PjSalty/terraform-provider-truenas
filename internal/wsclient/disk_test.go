package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"
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
