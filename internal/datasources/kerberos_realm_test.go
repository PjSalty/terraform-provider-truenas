package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewKerberosRealmDataSource(t *testing.T) {
	if NewKerberosRealmDataSource() == nil {
		t.Fatal("NewKerberosRealmDataSource returned nil")
	}
}

func TestKerberosRealmDataSource_Schema(t *testing.T) {
	ds := NewKerberosRealmDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "realm", "primary_kdc", "kdc", "admin_server", "kpasswd_server"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestKerberosRealmDataSource_Read_Success(t *testing.T) {
	primary := "kdc.example.com"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.KerberosRealm{
			ID:            3,
			Realm:         "EXAMPLE.COM",
			PrimaryKDC:    &primary,
			KDC:           []string{"kdc1.example.com", "kdc2.example.com"},
			AdminServer:   []string{"admin.example.com"},
			KPasswdServer: []string{"kpasswd.example.com"},
		})
	}))

	ds := NewKerberosRealmDataSource().(*KerberosRealmDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(3)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state KerberosRealmDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Realm.ValueString() != "EXAMPLE.COM" {
		t.Errorf("Realm: got %q", state.Realm.ValueString())
	}
	if state.PrimaryKDC.ValueString() != "kdc.example.com" {
		t.Errorf("PrimaryKDC: got %q", state.PrimaryKDC.ValueString())
	}
	if len(state.KDC.Elements()) != 2 {
		t.Errorf("KDC: got %d, want 2", len(state.KDC.Elements()))
	}
}

func TestKerberosRealmDataSource_Read_NoPrimaryKDC(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.KerberosRealm{
			ID:    1,
			Realm: "X",
			KDC:   []string{},
		})
	}))

	ds := NewKerberosRealmDataSource().(*KerberosRealmDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state KerberosRealmDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if !state.PrimaryKDC.IsNull() {
		t.Errorf("PrimaryKDC: expected null")
	}
}

func TestKerberosRealmDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewKerberosRealmDataSource().(*KerberosRealmDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestKerberosRealmDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
