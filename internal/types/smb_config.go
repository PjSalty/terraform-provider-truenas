package types

// SMB minimum-protocol values, matching middleware's
// SMBMinProtocol = Literal['SMB1', 'SMB2', 'SMB3'] (api/v26_0_0/smb.py:152).
const (
	SMBProtocolSMB1 = "SMB1"
	SMBProtocolSMB2 = "SMB2"
	SMBProtocolSMB3 = "SMB3"
)

// SMBDialect records which protocol key a given TrueNAS speaks.
//
// TrueNAS 26.0 replaced the boolean enable_smb1 with the tri-state
// minimum_protocol. Both sides are strict: the models are
// ConfigDict(extra="forbid", strict=True), so sending the wrong key is a
// hard ValidationError, not an ignored field. That cuts both ways, and
// minimum_protocol sent to a 25.10 box fails exactly as hard as
// enable_smb1 sent to a 26 box, so the client has to know which it is
// talking to rather than sending both and hoping.
type SMBDialect int

const (
	// SMBDialectUnknown means the shape has not been determined yet.
	SMBDialectUnknown SMBDialect = iota
	// SMBDialectEnableSMB1 is TrueNAS 25.10 and older.
	SMBDialectEnableSMB1
	// SMBDialectMinimumProtocol is TrueNAS 26.0 and newer.
	SMBDialectMinimumProtocol
)

// SMBConfig represents the SMB service configuration.
type SMBConfig struct {
	ID             int    `json:"id"`
	NetbiosName    string `json:"netbiosname"`
	Workgroup      string `json:"workgroup"`
	Description    string `json:"description"`
	UnixCharset    string `json:"unixcharset"`
	AAPLExtensions bool   `json:"aapl_extensions"`
	Guest          string `json:"guest"`
	Filemask       string `json:"filemask"`
	Dirmask        string `json:"dirmask"`

	// Exactly one of these two arrives, depending on the server version.
	// Both are pointers because a plain bool or string cannot tell
	// "the server sent false" apart from "the server never sent the key",
	// and that distinction is the whole version probe.
	EnableSMB1      *bool   `json:"enable_smb1"`
	MinimumProtocol *string `json:"minimum_protocol"`

	// SearchProtocols is new in TrueNAS 26.0 (macOS Spotlight). A pointer
	// so "this server has no such field" stays distinct from "the server
	// returned an empty list", which is its default.
	SearchProtocols *[]string `json:"search_protocols"`

	// Derived by NormalizeSMBConfig, never decoded from the wire.
	Protocol    string     `json:"-"` // SMB1 | SMB2 | SMB3
	SMB1Enabled bool       `json:"-"` // Protocol == SMB1
	Dialect     SMBDialect `json:"-"`
}

// SMBConfigUpdateRequest represents the request to update SMB configuration.
//
// MinimumProtocol is json:"-" on purpose: this struct is no longer the wire
// body. The client translates it into whichever protocol key the server
// actually speaks, so tagging it would let the 26-only key leak onto a 25.10
// request by accident.
type SMBConfigUpdateRequest struct {
	NetbiosName     *string `json:"netbiosname,omitempty"`
	Workgroup       *string `json:"workgroup,omitempty"`
	Description     *string `json:"description,omitempty"`
	UnixCharset     *string `json:"unixcharset,omitempty"`
	AAPLExtensions  *bool   `json:"aapl_extensions,omitempty"`
	Guest           *string `json:"guest,omitempty"`
	Filemask        *string `json:"filemask,omitempty"`
	Dirmask         *string `json:"dirmask,omitempty"`
	MinimumProtocol *string `json:"-"`

	// SearchProtocols is json:"-" for the same reason as MinimumProtocol:
	// this struct is not the wire body, and the key must not reach a
	// pre-26.0 server whose model forbids unknown fields.
	SearchProtocols *[]string `json:"-"`
}
