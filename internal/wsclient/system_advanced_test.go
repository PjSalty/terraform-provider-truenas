package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// systemAdvancedEntity is what system.advanced.config returns, with one
// remote server set.
func systemAdvancedEntity() map[string]interface{} {
	return map[string]interface{}{
		"id":                   1,
		"consolemenu":          false,
		"serialconsole":        false,
		"serialport":           "ttyS0",
		"serialspeed":          "9600",
		"powerdaemon":          false,
		"overprovision":        nil,
		"traceback":            true,
		"debugkernel":          false,
		"uploadcrash":          true,
		"anonstats":            true,
		"motd":                 "Welcome to TrueNAS",
		"login_banner":         "",
		"boot_scrub":           7,
		"fqdn_syslog":          false,
		"sysloglevel":          "F_INFO",
		"syslogservers":        []map[string]interface{}{{"host": "192.0.2.10", "transport": "UDP", "tls_certificate": nil}},
		"syslog_audit":         false,
		"kdump_enabled":        false,
		"kernel_extra_options": "",
		// Fields the provider does not manage must not break decoding.
		"advancedmode": false, "autotune": false, "consolemsg": true,
		"sed_user": "USER", "anonstats_token": "", "isolated_gpu_pci_ids": []string{},
	}
}

func systemAdvancedServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
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

func TestGetSystemAdvanced(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _ := systemAdvancedServer(t, "system.advanced.config", systemAdvancedEntity(), nil).NewClient(ctx)
	got, err := c.GetSystemAdvanced(ctx)
	if err != nil {
		t.Fatalf("GetSystemAdvanced: %v", err)
	}
	if got.Motd != "Welcome to TrueNAS" || got.BootScrub != 7 || got.Sysloglevel != "F_INFO" || got.Overprovision != nil {
		t.Errorf("decoded %+v", got)
	}
	if len(got.Syslogservers) != 1 || got.Syslogservers[0].Host != "192.0.2.10" ||
		got.Syslogservers[0].Transport != "UDP" || got.Syslogservers[0].TLSCertificate != nil {
		t.Errorf("syslogservers = %+v", got.Syslogservers)
	}
}

func TestUpdateSystemAdvanced(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("sends only what was set", func(t *testing.T) {
		var params []interface{}
		c, _ := systemAdvancedServer(t, "system.advanced.update", systemAdvancedEntity(), &params).NewClient(ctx)

		motd := "hello"
		if _, err := c.UpdateSystemAdvanced(ctx, &types.SystemAdvancedUpdateRequest{Motd: &motd}); err != nil {
			t.Fatalf("UpdateSystemAdvanced: %v", err)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		if body["motd"] != "hello" || len(body) != 1 {
			t.Errorf("body = %v, want only motd (an omitted key keeps the stored value)", body)
		}
	})

	// An empty list removes every server, so it must reach the wire, and a
	// non-TLS server sends tls_certificate as an explicit null.
	t.Run("sends an empty server list and a null certificate", func(t *testing.T) {
		var params []interface{}
		c, _ := systemAdvancedServer(t, "system.advanced.update", systemAdvancedEntity(), &params).NewClient(ctx)

		none := []types.SystemAdvancedSyslogServer{}
		if _, err := c.UpdateSystemAdvanced(ctx, &types.SystemAdvancedUpdateRequest{Syslogservers: &none}); err != nil {
			t.Fatalf("UpdateSystemAdvanced: %v", err)
		}
		body, _ := params[0].(map[string]interface{})
		list, ok := body["syslogservers"].([]interface{})
		if !ok || len(list) != 0 {
			t.Errorf("syslogservers = %#v, want []", body["syslogservers"])
		}

		one := []types.SystemAdvancedSyslogServer{{Host: "192.0.2.10:5514", Transport: "UDP"}}
		if _, err := c.UpdateSystemAdvanced(ctx, &types.SystemAdvancedUpdateRequest{Syslogservers: &one}); err != nil {
			t.Fatalf("UpdateSystemAdvanced: %v", err)
		}
		body, _ = params[0].(map[string]interface{})
		list, _ = body["syslogservers"].([]interface{})
		server, _ := list[0].(map[string]interface{})
		if cert, present := server["tls_certificate"]; !present || cert != nil {
			t.Errorf("server = %v, want tls_certificate null", server)
		}
	})
}

func TestSystemAdvanced_serverErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EINVAL] nope"}
	})
	c, _ := ts.NewClient(ctx)

	if _, err := c.GetSystemAdvanced(ctx); err == nil || !strings.Contains(err.Error(), "getting system advanced config") {
		t.Errorf("config: %v", err)
	}
	_, err := c.UpdateSystemAdvanced(ctx, &types.SystemAdvancedUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating system advanced config") {
		t.Errorf("update: %v", err)
	}
}

func TestSystemAdvanced_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := systemAdvancedServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}

	if _, err := bad("system.advanced.config").GetSystemAdvanced(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("config: %v", err)
	}
	_, err := bad("system.advanced.update").UpdateSystemAdvanced(ctx, &types.SystemAdvancedUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
}
