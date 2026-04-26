package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestGroupDataSource_Schema(t *testing.T) {
	ds := NewGroupDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "name", "gid", "smb", "builtin", "sudo_commands"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestGroupDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Group{
			{ID: 1, GID: 100, Name: "wheel", Builtin: true, SMB: false},
			{
				ID:           42,
				GID:          2000,
				Name:         "admins",
				Builtin:      false,
				SMB:          true,
				SudoCommands: []string{"/bin/ls", "/usr/bin/apt"},
			},
		})
	}))

	ds := NewGroupDataSource().(*GroupDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("admins")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state GroupDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueInt64() != 42 {
		t.Errorf("ID: got %d", state.ID.ValueInt64())
	}
	if state.GID.ValueInt64() != 2000 {
		t.Errorf("GID: got %d", state.GID.ValueInt64())
	}
	if state.SMB.ValueBool() != true {
		t.Errorf("SMB: got %v", state.SMB.ValueBool())
	}
	if state.SudoCommands.ValueString() != "/bin/ls,/usr/bin/apt" {
		t.Errorf("SudoCommands: got %q", state.SudoCommands.ValueString())
	}
}

func TestGroupDataSource_Read_EmptySudo(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Group{{ID: 1, Name: "users", GID: 100}})
	}))

	ds := NewGroupDataSource().(*GroupDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("users")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state GroupDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.SudoCommands.ValueString() != "" {
		t.Errorf("SudoCommands: got %q, want empty", state.SudoCommands.ValueString())
	}
}

func TestGroupDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Group{{ID: 1, Name: "other"}})
	}))

	ds := NewGroupDataSource().(*GroupDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestGroupDataSource_Read_ListError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewGroupDataSource().(*GroupDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("any")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}
