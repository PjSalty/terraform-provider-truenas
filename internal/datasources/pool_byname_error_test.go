package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// TestPoolDataSource_Read_ByName_ListError exercises the error branch in the
// by-name lookup path when the underlying ListPools API call fails.
func TestPoolDataSource_Read_ByName_ListError(t *testing.T) {
	c := newWSServer(t.Context(), t, wsError(wsclient.CodeMethodCallError, "[EFAULT] boom"))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"name": strVal("tank")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostic when ListPools fails")
	}
}
