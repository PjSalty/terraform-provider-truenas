package types

// ISCSIInitiator represents an iSCSI authorized initiator.
type ISCSIInitiator struct {
	ID         int      `json:"id"`
	Initiators []string `json:"initiators,omitempty"`
	Comment    string   `json:"comment,omitempty"`
}

// ISCSIInitiatorCreateRequest represents the request to create an iSCSI initiator.
type ISCSIInitiatorCreateRequest struct {
	Initiators []string `json:"initiators,omitempty"`
	Comment    string   `json:"comment,omitempty"`
}

// ISCSIInitiatorUpdateRequest represents the request to update an iSCSI initiator.
type ISCSIInitiatorUpdateRequest struct {
	Initiators []string `json:"initiators,omitempty"`
	Comment    string   `json:"comment,omitempty"`
}
