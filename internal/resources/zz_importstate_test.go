package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestReplicationResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &ReplicationResource{}
	sch := schemaOf(t, ctx, r)
	// Numeric: success
	reqResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "42"}, reqResp)
	if reqResp.Diagnostics.HasError() {
		t.Fatalf("ImportState(42): %v", reqResp.Diagnostics)
	}
	// Non-numeric: error branch
	reqResp2 := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "not-a-number"}, reqResp2)
	if !reqResp2.Diagnostics.HasError() {
		t.Errorf("expected error for non-numeric ID")
	}
}

func TestServiceResource_ImportState(t *testing.T) {
	ctx := context.Background()
	// Numeric: success (no API call)
	r := &ServiceResource{}
	sch := schemaOf(t, ctx, r)
	reqResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "5"}, reqResp)
	if reqResp.Diagnostics.HasError() {
		t.Fatalf("ImportState(5): %v", reqResp.Diagnostics)
	}

	// Non-numeric: looks up service by name via client
	// The by-name path queries services and picks the match, so the
	// handler answers with a LIST rather than a single entity.
	c := newWSTestClient(ctx, t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
		return []interface{}{
			map[string]interface{}{"id": 7, "service": "ssh", "enable": true, "state": "RUNNING"},
		}, nil
	})
	r2 := &ServiceResource{client: c}
	sch2 := schemaOf(t, ctx, r2)
	reqResp2 := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch2.Schema, Raw: rawFromValues(t, ctx, sch2, nil)}}
	r2.ImportState(ctx, resource.ImportStateRequest{ID: "ssh"}, reqResp2)
	_ = reqResp2.Diagnostics

	// Non-numeric with lookup failure
	c2 := newWSNotFoundClient(ctx, t)
	r3 := &ServiceResource{client: c2}
	sch3 := schemaOf(t, ctx, r3)
	reqResp3 := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch3.Schema, Raw: rawFromValues(t, ctx, sch3, nil)}}
	r3.ImportState(ctx, resource.ImportStateRequest{ID: "unknown"}, reqResp3)
	if !reqResp3.Diagnostics.HasError() {
		t.Errorf("expected error for unresolvable service name")
	}
}

var _ = tftypes.NewValue // keep import
