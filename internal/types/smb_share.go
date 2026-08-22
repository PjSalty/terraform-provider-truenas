package types

// SMBShare represents an SMB share in TrueNAS.
type SMBShare struct {
	ID        int    `json:"id"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Comment   string `json:"comment,omitempty"`
	Browsable bool   `json:"browsable"`
	ReadOnly  bool   `json:"readonly"`
	ABE       bool   `json:"access_based_share_enumeration"`
	Enabled   bool   `json:"enabled"`
	Purpose   string `json:"purpose,omitempty"`

	Options *SMBShareOptions `json:"options,omitempty"`
}

// SMBShareOptions carries the purpose-specific settings TrueNAS models as a
// discriminated union on the share's purpose. Every field is optional here
// because each one belongs to only some purposes, and sending one that does
// not belong is rejected outright ("Extra inputs are not permitted"), not
// ignored.
//
// The union's own discriminator is deliberately not sent. Middleware infers it
// from the share's purpose; verified against 26.0-BETA.1, where an options
// object with and without a nested purpose were both accepted identically.
type SMBShareOptions struct {
	// EXTERNAL_SHARE. Required for it, rejected everywhere else. Each entry
	// is SERVER\SHARE.
	RemotePath *[]string `json:"remote_path,omitempty"`

	// Every purpose except EXTERNAL_SHARE.
	HostsAllow *[]string `json:"hostsallow,omitempty"`
	HostsDeny  *[]string `json:"hostsdeny,omitempty"`

	// DEFAULT, MULTIPROTOCOL, TIME_LOCKED, PRIVATE_DATASETS, LEGACY, FCP.
	AAPLNameMangling *bool `json:"aapl_name_mangling,omitempty"`

	// TIMEMACHINE_SHARE.
	TimeMachineQuota    *int64 `json:"timemachine_quota,omitempty"`
	AutoSnapshot        *bool  `json:"auto_snapshot,omitempty"`
	AutoDatasetCreation *bool  `json:"auto_dataset_creation,omitempty"`

	// TIMEMACHINE_SHARE and PRIVATE_DATASETS_SHARE.
	DatasetNamingSchema *string `json:"dataset_naming_schema,omitempty"`

	// PRIVATE_DATASETS_SHARE, in gibibytes.
	AutoQuota *int64 `json:"auto_quota,omitempty"`

	// TIME_LOCKED_SHARE, in seconds, 60 to 15552000.
	GracePeriod *int64 `json:"grace_period,omitempty"`

	// TIMEMACHINE_SHARE and LEGACY_SHARE.
	VUID *string `json:"vuid,omitempty"`
}

// SMBShareCreateRequest represents the request to create an SMB share.
type SMBShareCreateRequest struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Comment   string `json:"comment,omitempty"`
	Browsable bool   `json:"browsable"`
	ReadOnly  bool   `json:"readonly"`
	ABE       bool   `json:"access_based_share_enumeration"`
	Enabled   bool   `json:"enabled"`
	Purpose   string `json:"purpose,omitempty"`

	Options *SMBShareOptions `json:"options,omitempty"`
}

// SMBShareUpdateRequest represents the request to update an SMB share.
type SMBShareUpdateRequest struct {
	Path      string `json:"path,omitempty"`
	Name      string `json:"name,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Browsable *bool  `json:"browsable,omitempty"`
	ReadOnly  *bool  `json:"readonly,omitempty"`
	ABE       *bool  `json:"access_based_share_enumeration,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
	Purpose   string `json:"purpose,omitempty"`

	Options *SMBShareOptions `json:"options,omitempty"`
}
