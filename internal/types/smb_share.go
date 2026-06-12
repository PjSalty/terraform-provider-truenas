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
}
