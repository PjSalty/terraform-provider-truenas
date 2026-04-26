package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewISCSIInitiatorDataSource(t *testing.T) {
	if NewISCSIInitiatorDataSource() == nil {
		t.Fatal("nil")
	}
}

func TestISCSIInitiatorDataSource_Schema(t *testing.T) {
	ds := NewISCSIInitiatorDataSource()
	resp := getDataSourceSchema(t, ds)
	for _, want := range []string{"id", "initiators", "comment"} {
		if _, ok := resp.Schema.GetAttributes()[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestISCSIInitiatorDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/iscsi/initiator/id/2" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.ISCSIInitiator{
			ID:         2,
			Initiators: []string{"iqn.2024-01.com.example:host1"},
			Comment:    "k8s nodes",
		})
	}))

	ds := NewISCSIInitiatorDataSource().(*ISCSIInitiatorDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(2)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state ISCSIInitiatorDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Comment.ValueString() != "k8s nodes" {
		t.Errorf("Comment: %q", state.Comment.ValueString())
	}
	if state.Initiators.IsNull() {
		t.Fatal("expected initiators populated")
	}
}

func TestISCSIInitiatorDataSource_Read_Empty(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, client.ISCSIInitiator{ID: 1})
	}))
	ds := NewISCSIInitiatorDataSource().(*ISCSIInitiatorDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
}

func TestISCSIInitiatorDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	ds := NewISCSIInitiatorDataSource().(*ISCSIInitiatorDataSource)
	ds.client = c
	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestISCSIInitiatorDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
