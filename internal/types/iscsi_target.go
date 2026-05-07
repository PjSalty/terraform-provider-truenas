package types

// ISCSITarget represents an iSCSI target.
type ISCSITarget struct {
	ID     int                `json:"id"`
	Name   string             `json:"name"`
	Alias  string             `json:"alias,omitempty"`
	Mode   string             `json:"mode"`
	Groups []ISCSITargetGroup `json:"groups,omitempty"`
}

// ISCSITargetGroup represents an iSCSI target group.
type ISCSITargetGroup struct {
	Portal     int    `json:"portal"`
	Initiator  int    `json:"initiator"`
	AuthMethod string `json:"authmethod"`
	Auth       int    `json:"auth"`
}

// ISCSITargetCreateRequest represents the request to create an iSCSI target.
type ISCSITargetCreateRequest struct {
	Name   string             `json:"name"`
	Alias  string             `json:"alias,omitempty"`
	Mode   string             `json:"mode"`
	Groups []ISCSITargetGroup `json:"groups,omitempty"`
}

// ISCSITargetUpdateRequest represents the request to update an iSCSI target.
type ISCSITargetUpdateRequest struct {
	Name   string             `json:"name,omitempty"`
	Alias  string             `json:"alias,omitempty"`
	Mode   string             `json:"mode,omitempty"`
	Groups []ISCSITargetGroup `json:"groups,omitempty"`
}
