package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestNewAPIKeyDataSource(t *testing.T) {
	if NewAPIKeyDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestAPIKeyDataSource_Schema(t *testing.T) {
	ds := NewAPIKeyDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	for _, want := range []string{"id", "name", "username", "local", "revoked"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestAPIKeyDataSource_Read_Success(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.APIKey{
		ID:       8,
		Name:     "terraform",
		Username: "root",
		Local:    true,
		Revoked:  false,
	}))

	ds := NewAPIKeyDataSource().(*APIKeyDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(8)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state APIKeyDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "terraform" {
		t.Errorf("Name: %q", state.Name.ValueString())
	}
	if state.Username.ValueString() != "root" {
		t.Errorf("Username: %q", state.Username.ValueString())
	}
	if !state.Local.ValueBool() {
		t.Errorf("Local: false")
	}
}

func TestAPIKeyDataSource_Read_NotFound(t *testing.T) {
	c := newWSServer(t.Context(), t, wsError(wsclient.CodeMethodCallError, "[ENOENT] nope"))
	ds := NewAPIKeyDataSource().(*APIKeyDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}
