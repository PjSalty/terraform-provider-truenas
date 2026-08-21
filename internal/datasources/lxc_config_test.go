package datasources

import (
	"context"
	"testing"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestNewLXCConfigDataSource(t *testing.T) {
	if NewLXCConfigDataSource() == nil {
		t.Fatal("NewLXCConfigDataSource returned nil")
	}
}

func TestLXCConfigDataSource_Schema(t *testing.T) {
	ds := NewLXCConfigDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "preferred_pool", "bridge", "v4_network", "v6_network"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestLXCConfigDataSource_Read_Success(t *testing.T) {
	pool := "tank"
	c := newWSServer(t.Context(), t, wsReturn(truenas.LXCConfig{
		ID:            1,
		PreferredPool: &pool,
		Bridge:        nil,
		V4Network:     "172.200.0.0/24",
		V6Network:     "fd42:4c58:43ae::/64",
	}))

	ds := NewLXCConfigDataSource().(*LXCConfigDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state LXCConfigDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "lxc_config" {
		t.Errorf("id = %q", state.ID.ValueString())
	}
	if state.PreferredPool.ValueString() != "tank" {
		t.Errorf("preferred_pool = %q", state.PreferredPool.ValueString())
	}
	// A null bridge must surface as a known empty string, not null, or a
	// consumer interpolating it has to guard against an unknown value.
	if state.Bridge.IsNull() || state.Bridge.IsUnknown() || state.Bridge.ValueString() != "" {
		t.Errorf("null bridge read back as %v", state.Bridge)
	}
	if state.V4Network.ValueString() != "172.200.0.0/24" || state.V6Network.ValueString() != "fd42:4c58:43ae::/64" {
		t.Errorf("networks = %q / %q", state.V4Network.ValueString(), state.V6Network.ValueString())
	}
}

// A null preferred_pool is the shipped default: it must not read back as
// null either.
func TestLXCConfigDataSource_Read_NullPool(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.LXCConfig{ID: 1}))
	ds := NewLXCConfigDataSource().(*LXCConfigDataSource)
	ds.client = c

	resp := callRead(context.Background(), ds, buildConfig(t.Context(), t, ds, nil))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state LXCConfigDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.PreferredPool.IsNull() || state.PreferredPool.ValueString() != "" {
		t.Errorf("null preferred_pool read back as %v", state.PreferredPool)
	}
}

func TestLXCConfigDataSource_Read_Error(t *testing.T) {
	c := newWSServer(t.Context(), t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: m}
	})
	ds := NewLXCConfigDataSource().(*LXCConfigDataSource)
	ds.client = c

	resp := callRead(context.Background(), ds, buildConfig(t.Context(), t, ds, nil))
	if !resp.Diagnostics.HasError() {
		t.Fatal("a missing lxc namespace was treated as a successful read")
	}
}
