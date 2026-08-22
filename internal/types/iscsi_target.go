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
//
// Initiator and Auth are pointers because upstream declares both as
// `int | None` defaulting to null, and null is meaningful: a null initiator
// group means "allow any initiator". As plain ints they serialized an omitted
// value as 0, and there is no group with id 0, so the insert failed on a
// foreign key rather than on anything the practitioner could read:
//
//	(sqlite3.IntegrityError) FOREIGN KEY constraint failed
type ISCSITargetGroup struct {
	Portal     int    `json:"portal"`
	Initiator  *int   `json:"initiator"`
	AuthMethod string `json:"authmethod"`
	Auth       *int   `json:"auth"`
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
