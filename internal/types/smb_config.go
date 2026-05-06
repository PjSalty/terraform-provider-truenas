package types

// SMBConfig represents the SMB service configuration.
type SMBConfig struct {
	ID             int    `json:"id"`
	NetbiosName    string `json:"netbiosname"`
	Workgroup      string `json:"workgroup"`
	Description    string `json:"description"`
	EnableSMB1     bool   `json:"enable_smb1"`
	UnixCharset    string `json:"unixcharset"`
	AAPLExtensions bool   `json:"aapl_extensions"`
	Guest          string `json:"guest"`
	Filemask       string `json:"filemask"`
	Dirmask        string `json:"dirmask"`
}

// SMBConfigUpdateRequest represents the request to update SMB configuration.
type SMBConfigUpdateRequest struct {
	NetbiosName    *string `json:"netbiosname,omitempty"`
	Workgroup      *string `json:"workgroup,omitempty"`
	Description    *string `json:"description,omitempty"`
	EnableSMB1     *bool   `json:"enable_smb1,omitempty"`
	UnixCharset    *string `json:"unixcharset,omitempty"`
	AAPLExtensions *bool   `json:"aapl_extensions,omitempty"`
	Guest          *string `json:"guest,omitempty"`
	Filemask       *string `json:"filemask,omitempty"`
	Dirmask        *string `json:"dirmask,omitempty"`
}
