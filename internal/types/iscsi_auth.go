package types

// ISCSIAuth represents an iSCSI CHAP authentication credential set.
type ISCSIAuth struct {
	ID            int    `json:"id"`
	Tag           int    `json:"tag"`
	User          string `json:"user"`
	Secret        string `json:"secret"`
	Peeruser      string `json:"peeruser"`
	Peersecret    string `json:"peersecret"`
	DiscoveryAuth string `json:"discovery_auth"`
}

// ISCSIAuthCreateRequest is the create payload.
type ISCSIAuthCreateRequest struct {
	Tag           int    `json:"tag"`
	User          string `json:"user"`
	Secret        string `json:"secret"`
	Peeruser      string `json:"peeruser,omitempty"`
	Peersecret    string `json:"peersecret,omitempty"`
	DiscoveryAuth string `json:"discovery_auth,omitempty"`
}

// ISCSIAuthUpdateRequest is the update payload.
type ISCSIAuthUpdateRequest struct {
	Tag           *int    `json:"tag,omitempty"`
	User          *string `json:"user,omitempty"`
	Secret        *string `json:"secret,omitempty"`
	Peeruser      *string `json:"peeruser,omitempty"`
	Peersecret    *string `json:"peersecret,omitempty"`
	DiscoveryAuth *string `json:"discovery_auth,omitempty"`
}
