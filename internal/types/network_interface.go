package types

// NetworkInterface represents a network interface as returned by /interface.
// Only the fields we surface in Terraform state are declared; additional
// fields are ignored via the absence of strict parsing (API returns a lot
// of state/runtime data we don't care about on the client side).
type NetworkInterface struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	Type                string                  `json:"type"`
	Description         string                  `json:"description"`
	IPv4DHCP            bool                    `json:"ipv4_dhcp"`
	IPv6Auto            bool                    `json:"ipv6_auto"`
	MTU                 *int                    `json:"mtu"`
	Aliases             []NetworkInterfaceAlias `json:"aliases"`
	BridgeMembers       []string                `json:"bridge_members"`
	LagProtocol         string                  `json:"lag_protocol"`
	LagPorts            []string                `json:"lag_ports"`
	VlanParentInterface string                  `json:"vlan_parent_interface"`
	VlanTag             *int                    `json:"vlan_tag"`
	VlanPCP             *int                    `json:"vlan_pcp"`
}

// NetworkInterfaceAlias represents an IP alias on an interface.
type NetworkInterfaceAlias struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Netmask int    `json:"netmask"`
}

// NetworkInterfaceCreateRequest is the POST /interface payload.
type NetworkInterfaceCreateRequest struct {
	Name                string                  `json:"name,omitempty"`
	Description         string                  `json:"description,omitempty"`
	Type                string                  `json:"type"`
	IPv4DHCP            bool                    `json:"ipv4_dhcp"`
	IPv6Auto            bool                    `json:"ipv6_auto"`
	Aliases             []NetworkInterfaceAlias `json:"aliases,omitempty"`
	MTU                 *int                    `json:"mtu,omitempty"`
	BridgeMembers       []string                `json:"bridge_members,omitempty"`
	LagProtocol         string                  `json:"lag_protocol,omitempty"`
	LagPorts            []string                `json:"lag_ports,omitempty"`
	VlanParentInterface string                  `json:"vlan_parent_interface,omitempty"`
	VlanTag             *int                    `json:"vlan_tag,omitempty"`
	VlanPCP             *int                    `json:"vlan_pcp,omitempty"`
}

// NetworkInterfaceUpdateRequest is the PUT /interface/id/{id_} payload.
// Mirrors the create request; all fields are optional for partial updates.
type NetworkInterfaceUpdateRequest struct {
	Description         *string                 `json:"description,omitempty"`
	IPv4DHCP            *bool                   `json:"ipv4_dhcp,omitempty"`
	IPv6Auto            *bool                   `json:"ipv6_auto,omitempty"`
	Aliases             []NetworkInterfaceAlias `json:"aliases,omitempty"`
	MTU                 *int                    `json:"mtu,omitempty"`
	BridgeMembers       []string                `json:"bridge_members,omitempty"`
	LagProtocol         *string                 `json:"lag_protocol,omitempty"`
	LagPorts            []string                `json:"lag_ports,omitempty"`
	VlanParentInterface *string                 `json:"vlan_parent_interface,omitempty"`
	VlanTag             *int                    `json:"vlan_tag,omitempty"`
	VlanPCP             *int                    `json:"vlan_pcp,omitempty"`
}

// InterfaceCommitRequest is the POST /interface/commit payload.
type InterfaceCommitRequest struct {
	Rollback       bool `json:"rollback"`
	CheckinTimeout int  `json:"checkin_timeout"`
}
