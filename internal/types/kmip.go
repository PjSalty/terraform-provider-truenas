package types

// KMIPConfig represents the singleton KMIP configuration.
type KMIPConfig struct {
	ID                   int     `json:"id"`
	Enabled              bool    `json:"enabled"`
	ManageSEDDisks       bool    `json:"manage_sed_disks"`
	ManageZFSKeys        bool    `json:"manage_zfs_keys"`
	Certificate          *int    `json:"certificate"`
	CertificateAuthority *int    `json:"certificate_authority"`
	Port                 int     `json:"port"`
	Server               *string `json:"server"`
	SSLVersion           string  `json:"ssl_version"`
}

// KMIPUpdateRequest is the singleton update payload.
type KMIPUpdateRequest struct {
	Enabled              *bool   `json:"enabled,omitempty"`
	ManageSEDDisks       *bool   `json:"manage_sed_disks,omitempty"`
	ManageZFSKeys        *bool   `json:"manage_zfs_keys,omitempty"`
	Certificate          *int    `json:"certificate,omitempty"`
	CertificateAuthority *int    `json:"certificate_authority,omitempty"`
	Port                 *int    `json:"port,omitempty"`
	Server               *string `json:"server,omitempty"`
	SSLVersion           *string `json:"ssl_version,omitempty"`
	ForceClear           *bool   `json:"force_clear,omitempty"`
	ChangeServer         *bool   `json:"change_server,omitempty"`
	Validate             *bool   `json:"validate,omitempty"`
}
