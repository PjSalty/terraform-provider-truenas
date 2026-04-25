package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewPrivilegeDataSource(t *testing.T) {
	if NewPrivilegeDataSource() == nil {
		t.Fatal("NewPrivilegeDataSource returned nil")
	}
}

func TestPrivilegeDataSource_Schema(t *testing.T) {
	ds := NewPrivilegeDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "builtin_name", "local_groups", "ds_groups", "roles", "web_shell",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestPrivilegeDataSource_Read_Success(t *testing.T) {
	builtin := "FULL_ADMIN"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.Privilege{
			ID:          2,
			Name:        "Admins",
			BuiltinName: &builtin,
			LocalGroups: []client.PrivilegeGroup{
				{ID: 1, GID: 544, Name: "wheel"},
				{ID: 2, GID: 545, Name: "admins"},
			},
			DSGroups: []interface{}{"S-1-5-32-544", float64(1000)},
			Roles:    []string{"READONLY_ADMIN", "SHARING_ADMIN"},
			WebShell: true,
		})
	}))

	ds := NewPrivilegeDataSource().(*PrivilegeDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(2)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state PrivilegeDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "Admins" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.BuiltinName.ValueString() != "FULL_ADMIN" {
		t.Errorf("BuiltinName: got %q", state.BuiltinName.ValueString())
	}
	if len(state.LocalGroups.Elements()) != 2 {
		t.Errorf("LocalGroups: got %d, want 2", len(state.LocalGroups.Elements()))
	}
	if len(state.DSGroups.Elements()) != 2 {
		t.Errorf("DSGroups: got %d, want 2", len(state.DSGroups.Elements()))
	}
	if len(state.Roles.Elements()) != 2 {
		t.Errorf("Roles: got %d, want 2", len(state.Roles.Elements()))
	}
	if !state.WebShell.ValueBool() {
		t.Errorf("WebShell: got %v", state.WebShell.ValueBool())
	}
}

func TestPrivilegeDataSource_Read_NoBuiltin(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.Privilege{
			ID:       5,
			Name:     "Custom",
			Roles:    []string{},
			DSGroups: []interface{}{},
		})
	}))

	ds := NewPrivilegeDataSource().(*PrivilegeDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(5)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state PrivilegeDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if !state.BuiltinName.IsNull() {
		t.Errorf("BuiltinName: expected null")
	}
}

func TestPrivilegeDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewPrivilegeDataSource().(*PrivilegeDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestPrivilegeDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
