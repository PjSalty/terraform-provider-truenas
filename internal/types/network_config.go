package types

// NetworkConfig represents the network configuration in TrueNAS.
type NetworkConfig struct {
	ID          int    `json:"id"`
	Hostname    string `json:"hostname"`
	Domain      string `json:"domain"`
	Nameserver1 string `json:"nameserver1"`
	Nameserver2 string `json:"nameserver2"`
	Nameserver3 string `json:"nameserver3"`
	IPv4Gateway string `json:"ipv4gateway"`
	HTTPProxy   string `json:"httpproxy"`
}

// NetworkConfigUpdateRequest represents the request to update network configuration.
type NetworkConfigUpdateRequest struct {
	Nameserver1 *string `json:"nameserver1,omitempty"`
	Nameserver2 *string `json:"nameserver2,omitempty"`
	Nameserver3 *string `json:"nameserver3,omitempty"`
}

// FullNetworkConfig represents the full network configuration in TrueNAS.
type FullNetworkConfig struct {
	ID          int      `json:"id"`
	Hostname    string   `json:"hostname"`
	Domain      string   `json:"domain"`
	IPv4Gateway string   `json:"ipv4gateway"`
	IPv6Gateway string   `json:"ipv6gateway"`
	Nameserver1 string   `json:"nameserver1"`
	Nameserver2 string   `json:"nameserver2"`
	Nameserver3 string   `json:"nameserver3"`
	HTTPProxy   string   `json:"httpproxy"`
	Hosts       []string `json:"hosts"`
}

// FullNetworkConfigUpdateRequest represents the request to update the full network configuration.
type FullNetworkConfigUpdateRequest struct {
	Hostname    *string  `json:"hostname,omitempty"`
	Domain      *string  `json:"domain,omitempty"`
	IPv4Gateway *string  `json:"ipv4gateway,omitempty"`
	IPv6Gateway *string  `json:"ipv6gateway,omitempty"`
	Nameserver1 *string  `json:"nameserver1,omitempty"`
	Nameserver2 *string  `json:"nameserver2,omitempty"`
	Nameserver3 *string  `json:"nameserver3,omitempty"`
	HTTPProxy   *string  `json:"httpproxy,omitempty"`
	Hosts       []string `json:"hosts,omitempty"`
}
