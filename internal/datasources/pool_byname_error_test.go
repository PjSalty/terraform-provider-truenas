package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestPoolDataSource_Read_ByName_ListError exercises the error branch in the
// by-name lookup path when the underlying ListPools API call fails.
func TestPoolDataSource_Read_ByName_ListError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewPoolDataSource().(*PoolDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("tank")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostic when ListPools fails")
	}
}
