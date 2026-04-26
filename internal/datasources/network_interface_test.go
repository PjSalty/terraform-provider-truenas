package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNewNetworkInterfaceDataSource(t *testing.T) {
	if NewNetworkInterfaceDataSource() == nil {
		t.Fatal("NewNetworkInterfaceDataSource returned nil")
	}
}

func TestNetworkInterfaceDataSource_Schema(t *testing.T) {
	ds := NewNetworkInterfaceDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "type", "description", "ipv4_dhcp", "ipv6_auto",
		"mtu", "bridge_members", "lag_protocol", "lag_ports",
		"vlan_parent_interface", "vlan_tag", "vlan_pcp", "aliases",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestNetworkInterfaceDataSource_Read_Physical(t *testing.T) {
	mtu := 9000
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.NetworkInterface{
			ID:          "eth0",
			Name:        "eth0",
			Type:        "PHYSICAL",
			Description: "wan uplink",
			IPv4DHCP:    false,
			IPv6Auto:    false,
			MTU:         &mtu,
			Aliases: []client.NetworkInterfaceAlias{
				{Type: "INET", Address: "192.168.1.10", Netmask: 24},
				{Type: "INET6", Address: "2001:db8::1", Netmask: 64},
			},
		})
	}))

	ds := NewNetworkInterfaceDataSource().(*NetworkInterfaceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("eth0")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state NetworkInterfaceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Type.ValueString() != "PHYSICAL" {
		t.Errorf("Type: got %q", state.Type.ValueString())
	}
	if state.MTU.ValueInt64() != 9000 {
		t.Errorf("MTU: got %d", state.MTU.ValueInt64())
	}
	if len(state.Aliases.Elements()) != 2 {
		t.Errorf("Aliases: got %d, want 2", len(state.Aliases.Elements()))
	}
}

func TestNetworkInterfaceDataSource_Read_VLAN(t *testing.T) {
	tag := 100
	pcp := 3
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.NetworkInterface{
			ID:                  "vlan100",
			Name:                "vlan100",
			Type:                "VLAN",
			VlanParentInterface: "eth0",
			VlanTag:             &tag,
			VlanPCP:             &pcp,
		})
	}))

	ds := NewNetworkInterfaceDataSource().(*NetworkInterfaceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("vlan100")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state NetworkInterfaceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.VlanTag.ValueInt64() != 100 {
		t.Errorf("VlanTag: got %d", state.VlanTag.ValueInt64())
	}
	if state.VlanParentInterface.ValueString() != "eth0" {
		t.Errorf("VlanParentInterface: got %q", state.VlanParentInterface.ValueString())
	}
}

func TestNetworkInterfaceDataSource_Read_Bridge(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.NetworkInterface{
			ID:            "br0",
			Name:          "br0",
			Type:          "BRIDGE",
			BridgeMembers: []string{"eth1", "eth2"},
		})
	}))

	ds := NewNetworkInterfaceDataSource().(*NetworkInterfaceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("br0")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state NetworkInterfaceDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if len(state.BridgeMembers.Elements()) != 2 {
		t.Errorf("BridgeMembers: got %d, want 2", len(state.BridgeMembers.Elements()))
	}
	if !state.MTU.IsNull() {
		t.Errorf("MTU: expected null")
	}
}

func TestNetworkInterfaceDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	ds := NewNetworkInterfaceDataSource().(*NetworkInterfaceDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"id": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestNetworkInterfaceDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
