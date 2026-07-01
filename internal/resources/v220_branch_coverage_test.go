package resources

// Branch coverage for the v2.2.0 port work, closing the CI tiered
// coverage gate (internal/resources has a 100% floor): the stat-fill
// branches of applyStatPreservingPlan, and the virtual Create aliases
// and create-error branches of the network interface resource.

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// stat fills only the values the plan did not know; planned values and
// mode are never overwritten.
func TestDirectoryResource_ApplyStatPreservingPlan_Branches(t *testing.T) {
	r := &DirectoryResource{}
	stat := &truenas.FilesystemStat{Mode: 0o40755, UID: 3000, GID: 4000}
	cases := []struct {
		name             string
		uid, gid         types.Int64
		wantUID, wantGID int64
	}{
		{"both null filled from stat", types.Int64Null(), types.Int64Null(), 3000, 4000},
		{"both unknown filled from stat", types.Int64Unknown(), types.Int64Unknown(), 3000, 4000},
		{"planned values win over stat", types.Int64Value(1000), types.Int64Value(1000), 1000, 1000},
		{"mixed: uid planned, gid from stat", types.Int64Value(1000), types.Int64Null(), 1000, 4000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &DirectoryResourceModel{
				Path: types.StringValue("/mnt/tank/d"),
				Mode: types.StringValue("755"),
				UID:  tc.uid,
				GID:  tc.gid,
			}
			r.applyStatPreservingPlan(stat, p)
			if p.UID.ValueInt64() != tc.wantUID || p.GID.ValueInt64() != tc.wantGID {
				t.Fatalf("uid/gid = %d/%d, want %d/%d", p.UID.ValueInt64(), p.GID.ValueInt64(), tc.wantUID, tc.wantGID)
			}
			if p.ID.ValueString() != "/mnt/tank/d" {
				t.Fatalf("id = %q, want the path", p.ID.ValueString())
			}
			if p.Mode.ValueString() != "755" {
				t.Fatalf("mode overwritten to %q; planned mode must win", p.Mode.ValueString())
			}
		})
	}
}

func v220BridgeBody() map[string]interface{} {
	return map[string]interface{}{
		"id": "br0", "name": "br0", "type": "BRIDGE",
		"description": "", "ipv4_dhcp": false, "ipv6_auto": false, "mtu": 1500,
		"state":            map[string]interface{}{"name": "br0"},
		"aliases":          []interface{}{map[string]interface{}{"type": "INET", "address": "10.10.30.40", "netmask": 24}},
		"failover_aliases": []interface{}{}, "failover_virtual_aliases": []interface{}{},
		"bridge_members": []interface{}{}, "lag_protocol": "", "lag_ports": []interface{}{},
		"vlan_parent_interface": "", "vlan_tag": nil, "vlan_pcp": nil,
	}
}

func v220AliasListValue() tftypes.Value {
	objT := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type": tftypes.String, "address": tftypes.String, "netmask": tftypes.Number,
	}}
	obj := tftypes.NewValue(objT, map[string]tftypes.Value{
		"type":    tftypes.NewValue(tftypes.String, "INET"),
		"address": tftypes.NewValue(tftypes.String, "10.10.30.40"),
		"netmask": tftypes.NewValue(tftypes.Number, int64(24)),
	})
	return tftypes.NewValue(tftypes.List{ElementType: objT}, []tftypes.Value{obj})
}

// a virtual create with a configured alias exercises the
// aliasesFromList ok branch of Create.
func TestNetworkInterfaceResource_Create_VirtualWithAliases(t *testing.T) {
	c := newWSJSONServerClient(t, v220BridgeBody())
	r := &NetworkInterfaceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":    str("br0"),
		"type":    str("BRIDGE"),
		"aliases": v220AliasListValue(),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create(BRIDGE with aliases): %v", cResp.Diagnostics)
	}
}

// a failing interface.create must surface as the Error Creating
// Interface diagnostic (the virtual-create error branch).
func TestNetworkInterfaceResource_Create_VirtualCreateError(t *testing.T) {
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		if method == "interface.create" {
			return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
		}
		return v220BridgeBody(), nil
	})
	c, err := ts.NewClient(context.Background())
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	r := &NetworkInterfaceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name": str("br0"),
		"type": str("BRIDGE"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Fatal("Create should error when interface.create fails")
	}
	found := false
	for _, d := range cResp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Error Creating Interface") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Error Creating Interface diagnostic, got: %v", cResp.Diagnostics)
	}
}
