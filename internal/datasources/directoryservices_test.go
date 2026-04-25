package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewDirectoryServicesDataSource(t *testing.T) {
	if NewDirectoryServicesDataSource() == nil {
		t.Fatal("NewDirectoryServicesDataSource returned nil")
	}
}

func TestDirectoryServicesDataSource_Schema(t *testing.T) {
	ds := NewDirectoryServicesDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "service_type", "enable", "enable_account_cache",
		"enable_dns_updates", "timeout", "kerberos_realm",
		"credential_json", "configuration_json",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestDirectoryServicesDataSource_Read_Enabled(t *testing.T) {
	svcType := "ACTIVEDIRECTORY"
	realm := "EXAMPLE.COM"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{
			ID:                 1,
			ServiceType:        &svcType,
			Enable:             true,
			EnableAccountCache: true,
			EnableDNSUpdates:   false,
			Timeout:            30,
			KerberosRealm:      &realm,
			Credential: map[string]interface{}{
				"type": "KERBEROS_USER",
				"user": "admin",
			},
			Configuration: map[string]interface{}{
				"domain": "example.com",
			},
		})
	}))

	ds := NewDirectoryServicesDataSource().(*DirectoryServicesDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state DirectoryServicesDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ServiceType.ValueString() != "ACTIVEDIRECTORY" {
		t.Errorf("ServiceType: got %q", state.ServiceType.ValueString())
	}
	if !state.Enable.ValueBool() {
		t.Errorf("Enable: got %v", state.Enable.ValueBool())
	}
	if state.Timeout.ValueInt64() != 30 {
		t.Errorf("Timeout: got %d", state.Timeout.ValueInt64())
	}
	if state.KerberosRealm.ValueString() != "EXAMPLE.COM" {
		t.Errorf("KerberosRealm: got %q", state.KerberosRealm.ValueString())
	}
	if !strings.Contains(state.CredentialJSON.ValueString(), "KERBEROS_USER") {
		t.Errorf("CredentialJSON missing type: %q", state.CredentialJSON.ValueString())
	}
	if !strings.Contains(state.ConfigurationJSON.ValueString(), "example.com") {
		t.Errorf("ConfigurationJSON missing domain: %q", state.ConfigurationJSON.ValueString())
	}
}

func TestDirectoryServicesDataSource_Read_Disabled(t *testing.T) {
	// No service type, no credential — should produce null strings.
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{
			ID:     1,
			Enable: false,
		})
	}))

	ds := NewDirectoryServicesDataSource().(*DirectoryServicesDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state DirectoryServicesDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if !state.ServiceType.IsNull() {
		t.Errorf("ServiceType: expected null, got %q", state.ServiceType.ValueString())
	}
	if !state.CredentialJSON.IsNull() {
		t.Errorf("CredentialJSON: expected null")
	}
}

func TestDirectoryServicesDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewDirectoryServicesDataSource().(*DirectoryServicesDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestDirectoryServicesDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
