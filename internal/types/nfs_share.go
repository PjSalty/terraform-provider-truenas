package types

// NFSShare represents an NFS share in TrueNAS.
type NFSShare struct {
	ID           int      `json:"id"`
	Path         string   `json:"path"`
	Aliases      []string `json:"aliases,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	ReadOnly     bool     `json:"ro"`
	MaprootUser  string   `json:"maproot_user,omitempty"`
	MaprootGroup string   `json:"maproot_group,omitempty"`
	MapallUser   string   `json:"mapall_user,omitempty"`
	MapallGroup  string   `json:"mapall_group,omitempty"`
	Security     []string `json:"security,omitempty"`
	Enabled      bool     `json:"enabled"`
	Networks     []string `json:"networks,omitempty"`
}

// NFSShareCreateRequest represents the request to create an NFS share.
type NFSShareCreateRequest struct {
	Path         string   `json:"path"`
	Aliases      []string `json:"aliases,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	ReadOnly     bool     `json:"ro"`
	MaprootUser  string   `json:"maproot_user,omitempty"`
	MaprootGroup string   `json:"maproot_group,omitempty"`
	MapallUser   string   `json:"mapall_user,omitempty"`
	MapallGroup  string   `json:"mapall_group,omitempty"`
	Security     []string `json:"security,omitempty"`
	Enabled      bool     `json:"enabled"`
	Networks     []string `json:"networks,omitempty"`
}

// NFSShareUpdateRequest represents the request to update an NFS share.
type NFSShareUpdateRequest struct {
	Path         string   `json:"path,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
	Comment      string   `json:"comment,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	ReadOnly     *bool    `json:"ro,omitempty"`
	MaprootUser  string   `json:"maproot_user,omitempty"`
	MaprootGroup string   `json:"maproot_group,omitempty"`
	MapallUser   string   `json:"mapall_user,omitempty"`
	MapallGroup  string   `json:"mapall_group,omitempty"`
	Security     []string `json:"security,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
	Networks     []string `json:"networks,omitempty"`
}
