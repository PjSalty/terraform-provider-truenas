package wsclient

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// iscsiGlobalEntity is what iscsi.global.config returns on a fresh
// TrueNAS 25.10 install.
func iscsiGlobalEntity() map[string]interface{} {
	return map[string]interface{}{
		"id":                   1,
		"basename":             "iqn.2005-10.org.freenas.ctl",
		"isns_servers":         []string{},
		"listen_port":          3260,
		"pool_avail_threshold": nil,
		"alua":                 false,
		"iser":                 false,
	}
}

func iscsiGlobalServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
	t.Helper()
	return NewTestServer(t, func(_ context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		if m != method {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		}
		if got != nil {
			*got = params
		}
		return result, nil
	})
}

func TestGetISCSIGlobal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("null threshold", func(t *testing.T) {
		c, _ := iscsiGlobalServer(t, "iscsi.global.config", iscsiGlobalEntity(), nil).NewClient(ctx)
		got, err := c.GetISCSIGlobal(ctx)
		if err != nil {
			t.Fatalf("GetISCSIGlobal: %v", err)
		}
		if got.ID != 1 || got.Basename != "iqn.2005-10.org.freenas.ctl" || got.ListenPort != 3260 || got.Alua || got.Iser {
			t.Errorf("decoded %+v", got)
		}
		if got.PoolAvailThreshold != nil || len(got.IsnsServers) != 0 {
			t.Errorf("threshold = %v, isns = %v", got.PoolAvailThreshold, got.IsnsServers)
		}
	})

	t.Run("set threshold and servers", func(t *testing.T) {
		entity := iscsiGlobalEntity()
		entity["pool_avail_threshold"] = 80
		entity["isns_servers"] = []string{"10.0.0.5", "10.0.0.6:3205"}
		c, _ := iscsiGlobalServer(t, "iscsi.global.config", entity, nil).NewClient(ctx)
		got, err := c.GetISCSIGlobal(ctx)
		if err != nil {
			t.Fatalf("GetISCSIGlobal: %v", err)
		}
		if got.PoolAvailThreshold == nil || *got.PoolAvailThreshold != 80 {
			t.Errorf("threshold = %v", got.PoolAvailThreshold)
		}
		if !slices.Equal(got.IsnsServers, []string{"10.0.0.5", "10.0.0.6:3205"}) {
			t.Errorf("isns = %v", got.IsnsServers)
		}
	})
}

func TestUpdateISCSIGlobal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// iscsi.global.update empties the iSNS list when the key is missing,
	// so the request must carry it even when nothing else does.
	t.Run("iSNS servers are always sent", func(t *testing.T) {
		var params []interface{}
		c, _ := iscsiGlobalServer(t, "iscsi.global.update", iscsiGlobalEntity(), &params).NewClient(ctx)

		if _, err := c.UpdateISCSIGlobal(ctx, &types.ISCSIGlobalUpdateRequest{IsnsServers: []string{}}); err != nil {
			t.Fatalf("UpdateISCSIGlobal: %v", err)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		servers, present := body["isns_servers"].([]interface{})
		if !present || len(servers) != 0 {
			t.Errorf("isns_servers = %#v, want an empty list", body["isns_servers"])
		}
		for _, k := range []string{"basename", "listen_port", "pool_avail_threshold"} {
			if _, sent := body[k]; sent {
				t.Errorf("%s sent for an update that did not set it: %v", k, body)
			}
		}
	})

	t.Run("sends every field", func(t *testing.T) {
		var params []interface{}
		c, _ := iscsiGlobalServer(t, "iscsi.global.update", iscsiGlobalEntity(), &params).NewClient(ctx)

		name, port := "iqn.2024-01.example:ctl", 3261
		req := &types.ISCSIGlobalUpdateRequest{
			Basename:           &name,
			IsnsServers:        []string{"10.0.0.5"},
			ListenPort:         &port,
			PoolAvailThreshold: types.Value(75),
		}
		if _, err := c.UpdateISCSIGlobal(ctx, req); err != nil {
			t.Fatalf("UpdateISCSIGlobal: %v", err)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		if body["basename"] != name || body["listen_port"] != float64(port) || body["pool_avail_threshold"] != float64(75) {
			t.Errorf("body = %v", body)
		}
	})

	t.Run("null threshold is sent as null", func(t *testing.T) {
		var params []interface{}
		c, _ := iscsiGlobalServer(t, "iscsi.global.update", iscsiGlobalEntity(), &params).NewClient(ctx)

		req := &types.ISCSIGlobalUpdateRequest{IsnsServers: []string{}, PoolAvailThreshold: types.Null[int]()}
		if _, err := c.UpdateISCSIGlobal(ctx, req); err != nil {
			t.Fatalf("UpdateISCSIGlobal: %v", err)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		v, present := body["pool_avail_threshold"]
		if !present || v != nil {
			t.Errorf("pool_avail_threshold = %v (present %v), want null", v, present)
		}
	})
}

func TestISCSIGlobal_serverErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EINVAL] nope"}
	})
	c, _ := ts.NewClient(ctx)

	if _, err := c.GetISCSIGlobal(ctx); err == nil || !strings.Contains(err.Error(), "getting iSCSI global config") {
		t.Errorf("config: %v", err)
	}
	_, err := c.UpdateISCSIGlobal(ctx, &types.ISCSIGlobalUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI global config") {
		t.Errorf("update: %v", err)
	}
}

func TestISCSIGlobal_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := iscsiGlobalServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}

	if _, err := bad("iscsi.global.config").GetISCSIGlobal(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("config: %v", err)
	}
	_, err := bad("iscsi.global.update").UpdateISCSIGlobal(ctx, &types.ISCSIGlobalUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
}
