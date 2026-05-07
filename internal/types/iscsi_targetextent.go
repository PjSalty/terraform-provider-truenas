package types

// ISCSITargetExtent represents an iSCSI target-to-extent mapping.
type ISCSITargetExtent struct {
	ID     int `json:"id"`
	Target int `json:"target"`
	Extent int `json:"extent"`
	LunID  int `json:"lunid"`
}

// ISCSITargetExtentCreateRequest represents the request to create a target-extent association.
type ISCSITargetExtentCreateRequest struct {
	Target int  `json:"target"`
	Extent int  `json:"extent"`
	LunID  *int `json:"lunid,omitempty"`
}

// ISCSITargetExtentUpdateRequest represents the request to update a target-extent association.
type ISCSITargetExtentUpdateRequest struct {
	Target int  `json:"target,omitempty"`
	Extent int  `json:"extent,omitempty"`
	LunID  *int `json:"lunid,omitempty"`
}
