package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewISCSITargetDataSource(t *testing.T) {
	if NewISCSITargetDataSource() == nil {
		t.Fatal("NewISCSITargetDataSource returned nil")
	}
}

func TestISCSITargetDataSource_Schema(t *testing.T) {
	ds := NewISCSITargetDataSource()
	resp := getDataSourceSchema(t, ds)
	for _, want := range []string{"id", "name", "alias", "mode", "groups"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestISCSITargetDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/iscsi/target/id/7" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.ISCSITarget{
			ID:    7,
			Name:  "iqn-test",
			Alias: "prod",
			Mode:  "ISCSI",
			Groups: []client.ISCSITargetGroup{
				{Portal: 1, Initiator: 2, AuthMethod: "CHAP", Auth: 3},
			},
		})
	}))

	ds := NewISCSITargetDataSource().(*ISCSITargetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(7)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state ISCSITargetDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "iqn-test" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.Alias.ValueString() != "prod" {
		t.Errorf("Alias: got %q", state.Alias.ValueString())
	}
	if state.Mode.ValueString() != "ISCSI" {
		t.Errorf("Mode: got %q", state.Mode.ValueString())
	}
	if state.Groups.IsNull() {
		t.Fatal("expected groups to be populated")
	}
}

func TestISCSITargetDataSource_Read_EmptyGroups(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, client.ISCSITarget{ID: 1, Name: "x", Mode: "ISCSI"})
	}))

	ds := NewISCSITargetDataSource().(*ISCSITargetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
}

func TestISCSITargetDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "missing"})
	}))

	ds := NewISCSITargetDataSource().(*ISCSITargetDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestISCSITargetDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
