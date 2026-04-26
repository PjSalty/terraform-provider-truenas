package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewKeychainCredentialDataSource(t *testing.T) {
	if NewKeychainCredentialDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestKeychainCredentialDataSource_Schema(t *testing.T) {
	ds := NewKeychainCredentialDataSource()
	resp := getDataSourceSchema(t, ds)
	for _, want := range []string{"id", "name", "type", "attributes_json"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestKeychainCredentialDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/keychaincredential/id/4" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.KeychainCredential{
			ID:   4,
			Name: "backup-key",
			Type: "SSH_KEY_PAIR",
			Attributes: map[string]interface{}{
				"public_key":  "ssh-rsa AAA...",
				"private_key": "-----BEGIN...",
			},
		})
	}))

	ds := NewKeychainCredentialDataSource().(*KeychainCredentialDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(4)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state KeychainCredentialDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "backup-key" {
		t.Errorf("Name: %q", state.Name.ValueString())
	}
	if state.Type.ValueString() != "SSH_KEY_PAIR" {
		t.Errorf("Type: %q", state.Type.ValueString())
	}
	if !strings.Contains(state.AttributesJSON.ValueString(), "public_key") {
		t.Errorf("AttributesJSON does not contain expected key: %s", state.AttributesJSON.ValueString())
	}
}

func TestKeychainCredentialDataSource_Read_NilAttributes(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, client.KeychainCredential{ID: 1, Name: "x", Type: "SSH_CREDENTIALS"})
	}))
	ds := NewKeychainCredentialDataSource().(*KeychainCredentialDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state KeychainCredentialDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.AttributesJSON.ValueString() != "{}" {
		t.Errorf("expected empty json, got %q", state.AttributesJSON.ValueString())
	}
}

func TestKeychainCredentialDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	ds := NewKeychainCredentialDataSource().(*KeychainCredentialDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestKeychainCredentialDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
