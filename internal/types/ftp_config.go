package types

// FTPConfig represents the FTP service configuration.
type FTPConfig struct {
	ID            int    `json:"id"`
	Port          int    `json:"port"`
	Clients       int    `json:"clients"`
	IPConnections int    `json:"ipconnections"`
	LoginAttempt  int    `json:"loginattempt"`
	Timeout       int    `json:"timeout"`
	OnlyAnonymous bool   `json:"onlyanonymous"`
	OnlyLocal     bool   `json:"onlylocal"`
	Banner        string `json:"banner"`
	Filemask      string `json:"filemask"`
	Dirmask       string `json:"dirmask"`
	FXP           bool   `json:"fxp"`
	Resume        bool   `json:"resume"`
	DefaultRoot   bool   `json:"defaultroot"`
	TLS           bool   `json:"tls"`
}

// FTPConfigUpdateRequest represents the request to update FTP configuration.
type FTPConfigUpdateRequest struct {
	Port          *int    `json:"port,omitempty"`
	Clients       *int    `json:"clients,omitempty"`
	IPConnections *int    `json:"ipconnections,omitempty"`
	LoginAttempt  *int    `json:"loginattempt,omitempty"`
	Timeout       *int    `json:"timeout,omitempty"`
	OnlyAnonymous *bool   `json:"onlyanonymous,omitempty"`
	OnlyLocal     *bool   `json:"onlylocal,omitempty"`
	Banner        *string `json:"banner,omitempty"`
	Filemask      *string `json:"filemask,omitempty"`
	Dirmask       *string `json:"dirmask,omitempty"`
	FXP           *bool   `json:"fxp,omitempty"`
	Resume        *bool   `json:"resume,omitempty"`
	DefaultRoot   *bool   `json:"defaultroot,omitempty"`
	TLS           *bool   `json:"tls,omitempty"`
}
