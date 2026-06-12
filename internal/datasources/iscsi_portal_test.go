package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestNewISCSIPortalDataSource(t *testing.T) {
	if NewISCSIPortalDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestISCSIPortalDataSource_Schema(t *testing.T) {
	ds := NewISCSIPortalDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	for _, want := range []string{"id", "comment", "tag", "listen"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestISCSIPortalDataSource_Read_Success(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.ISCSIPortal{
		ID:      3,
		Comment: "prod",
		Tag:     1,
		Listen:  []truenas.ISCSIPortalListen{{IP: "0.0.0.0", Port: 3260}},
	}))

	ds := NewISCSIPortalDataSource().(*ISCSIPortalDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(3)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state ISCSIPortalDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Comment.ValueString() != "prod" {
		t.Errorf("Comment: got %q", state.Comment.ValueString())
	}
	if state.Tag.ValueInt64() != 1 {
		t.Errorf("Tag: got %d", state.Tag.ValueInt64())
	}
	if state.Listen.IsNull() {
		t.Fatal("expected listen populated")
	}
}

func TestISCSIPortalDataSource_Read_EmptyListen(t *testing.T) {
	skipWSCutover(t)
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, truenas.ISCSIPortal{ID: 1})
	}))
	ds := NewISCSIPortalDataSource().(*ISCSIPortalDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
}

func TestISCSIPortalDataSource_Read_NotFound(t *testing.T) {
	skipWSCutover(t)
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	ds := NewISCSIPortalDataSource().(*ISCSIPortalDataSource)
	ds.client = c
	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestISCSIPortalDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
