package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --- Network Interface ---

func TestNetworkInterfaceResource_MapResponseToModel_Cases(t *testing.T) {
	r := &NetworkInterfaceResource{}
	ctx := context.Background()
	cases := []struct {
		name        string
		iface       *client.NetworkInterface
		wantID      string
		wantType    string
		wantDHCP    bool
		wantAliases int
		wantLag     int
	}{
		{
			name: "bridge interface",
			iface: &client.NetworkInterface{
				ID: "br0", Name: "br0", Type: "BRIDGE", Description: "br",
				BridgeMembers: []string{"eth0", "eth1"},
			},
			wantID: "br0", wantType: "BRIDGE",
		},
		{
			name: "vlan interface with static address",
			iface: &client.NetworkInterface{
				ID: "vlan10", Name: "vlan10", Type: "VLAN",
				Aliases: []client.NetworkInterfaceAlias{
					{Type: "INET", Address: "10.0.0.1", Netmask: 24},
				},
				VlanParentInterface: "eth0",
				VlanTag:             intPtr(10),
			},
			wantID: "vlan10", wantType: "VLAN", wantAliases: 1,
		},
		{
			name: "lag interface",
			iface: &client.NetworkInterface{
				ID: "bond0", Name: "bond0", Type: "LINK_AGGREGATION",
				LagProtocol: "LACP",
				LagPorts:    []string{"eth2", "eth3"},
			},
			wantID: "bond0", wantType: "LINK_AGGREGATION", wantLag: 2,
		},
		{
			name: "dhcp interface",
			iface: &client.NetworkInterface{
				ID: "eth0", Name: "eth0", Type: "PHYSICAL",
				IPv4DHCP: true, MTU: intPtr(1500),
			},
			wantID: "eth0", wantType: "PHYSICAL", wantDHCP: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m NetworkInterfaceResourceModel
			r.mapResponseToModel(ctx, tc.iface, &m)
			if m.ID.ValueString() != tc.wantID {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.Type.ValueString() != tc.wantType {
				t.Errorf("Type = %q", m.Type.ValueString())
			}
			if m.IPv4DHCP.ValueBool() != tc.wantDHCP {
				t.Errorf("IPv4DHCP = %v", m.IPv4DHCP.ValueBool())
			}
			if got := len(m.Aliases.Elements()); got != tc.wantAliases {
				t.Errorf("Aliases = %d, want %d", got, tc.wantAliases)
			}
			if got := len(m.LagPorts.Elements()); got != tc.wantLag {
				t.Errorf("LagPorts = %d, want %d", got, tc.wantLag)
			}
		})
	}
}

func TestNetworkInterfaceResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewNetworkInterfaceResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "type", "ipv4_dhcp", "aliases"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["type"].IsRequired() {
		t.Error("type should be required")
	}
}

// --- Static Route ---

func TestStaticRouteResource_MapResponseToModel_Cases(t *testing.T) {
	r := &StaticRouteResource{}
	cases := []struct {
		name  string
		route *client.StaticRoute
		want  StaticRouteResourceModel
	}{
		{name: "basic route", route: &client.StaticRoute{ID: 1, Destination: "10.0.0.0/8", Gateway: "192.168.1.1", Description: "office"}},
		{name: "ipv6 route", route: &client.StaticRoute{ID: 2, Destination: "fd00::/8", Gateway: "fe80::1"}},
		{name: "default route", route: &client.StaticRoute{ID: 3, Destination: "0.0.0.0/0", Gateway: "192.168.1.254"}},
		{name: "no description", route: &client.StaticRoute{ID: 4, Destination: "172.16.0.0/12", Gateway: "10.1.1.1"}},
	}
	for _, tc := range cases {
		_ = tc.want
		t.Run(tc.name, func(t *testing.T) {
			var m StaticRouteResourceModel
			r.mapResponseToModel(tc.route, &m)
			if m.Destination.ValueString() != tc.route.Destination {
				t.Errorf("Destination mismatch")
			}
			if m.Gateway.ValueString() != tc.route.Gateway {
				t.Errorf("Gateway mismatch")
			}
			if m.Description.ValueString() != tc.route.Description {
				t.Errorf("Description mismatch")
			}
		})
	}
}

func TestStaticRouteResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewStaticRouteResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "destination", "gateway", "description"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["destination"].IsRequired() {
		t.Error("destination should be required")
	}
	if !attrs["gateway"].IsRequired() {
		t.Error("gateway should be required")
	}
}

// --- DNS Nameserver ---

func TestDNSNameserverResource_MapResponseToModel_Cases(t *testing.T) {
	r := &DNSNameserverResource{}
	cases := []struct {
		name string
		cfg  *client.NetworkConfig
	}{
		{name: "three nameservers", cfg: &client.NetworkConfig{Nameserver1: "1.1.1.1", Nameserver2: "8.8.8.8", Nameserver3: "9.9.9.9"}},
		{name: "single nameserver", cfg: &client.NetworkConfig{Nameserver1: "10.0.0.1"}},
		{name: "all empty", cfg: &client.NetworkConfig{}},
		{name: "ipv6 nameserver", cfg: &client.NetworkConfig{Nameserver1: "2606:4700:4700::1111"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m DNSNameserverResourceModel
			r.mapResponseToModel(tc.cfg, &m)
			if m.ID.ValueString() != "network_config" {
				t.Errorf("ID = %q", m.ID.ValueString())
			}
			if m.Nameserver1.ValueString() != tc.cfg.Nameserver1 {
				t.Errorf("Nameserver1 mismatch")
			}
			if m.Nameserver2.ValueString() != tc.cfg.Nameserver2 {
				t.Errorf("Nameserver2 mismatch")
			}
			if m.Nameserver3.ValueString() != tc.cfg.Nameserver3 {
				t.Errorf("Nameserver3 mismatch")
			}
		})
	}
}

func TestDNSNameserverResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewDNSNameserverResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "nameserver1", "nameserver2", "nameserver3"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
}

// --- Tunable ---

func TestTunableResource_MapResponseToModel_Cases(t *testing.T) {
	r := &TunableResource{}
	cases := []struct {
		name    string
		tunable *client.Tunable
	}{
		{name: "sysctl tunable", tunable: &client.Tunable{ID: 1, Type: "SYSCTL", Var: "net.ipv4.tcp_keepalive_time", Value: "60", Enabled: true}},
		{name: "udev tunable", tunable: &client.Tunable{ID: 2, Type: "UDEV", Var: "test", Value: "val", Comment: "udev test"}},
		{name: "disabled tunable", tunable: &client.Tunable{ID: 3, Type: "SYSCTL", Var: "x", Value: "1", Enabled: false}},
		{name: "zfs tunable", tunable: &client.Tunable{ID: 4, Type: "ZFS", Var: "zfs_arc_max", Value: "1073741824", Enabled: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m TunableResourceModel
			r.mapResponseToModel(tc.tunable, &m)
			if m.Type.ValueString() != tc.tunable.Type {
				t.Errorf("Type mismatch")
			}
			if m.Var.ValueString() != tc.tunable.Var {
				t.Errorf("Var mismatch")
			}
			if m.Value.ValueString() != tc.tunable.Value {
				t.Errorf("Value mismatch")
			}
			if m.Enabled.ValueBool() != tc.tunable.Enabled {
				t.Errorf("Enabled mismatch")
			}
		})
	}
}

func TestTunableResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewTunableResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "type", "var", "value", "enabled"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["var"].IsRequired() {
		t.Error("var should be required")
	}
}

// --- Init Script ---

func TestInitScriptResource_MapResponseToModel_Cases(t *testing.T) {
	r := &InitScriptResource{}
	cases := []struct {
		name   string
		script *client.InitScript
	}{
		{name: "command prejenkins", script: &client.InitScript{ID: 1, Type: "COMMAND", Command: "echo hi", When: "POSTINIT", Enabled: true, Timeout: 10}},
		{name: "script preinit", script: &client.InitScript{ID: 2, Type: "SCRIPT", Script: "/mnt/tank/init.sh", When: "PREINIT", Enabled: true, Timeout: 30}},
		{name: "disabled script", script: &client.InitScript{ID: 3, Type: "COMMAND", Command: "x", When: "POSTINIT", Enabled: false}},
		{name: "shutdown script", script: &client.InitScript{ID: 4, Type: "COMMAND", Command: "sync", When: "SHUTDOWN", Enabled: true, Comment: "sync"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m InitScriptResourceModel
			r.mapResponseToModel(tc.script, &m)
			if m.Type.ValueString() != tc.script.Type {
				t.Errorf("Type mismatch")
			}
			if m.Command.ValueString() != tc.script.Command {
				t.Errorf("Command mismatch")
			}
			if m.When.ValueString() != tc.script.When {
				t.Errorf("When mismatch")
			}
			if m.Enabled.ValueBool() != tc.script.Enabled {
				t.Errorf("Enabled mismatch")
			}
		})
	}
}

func TestInitScriptResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewInitScriptResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "type", "when", "enabled"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["type"].IsRequired() {
		t.Error("type should be required")
	}
}
