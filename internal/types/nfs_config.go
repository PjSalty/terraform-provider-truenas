package types

// NFSConfig represents the NFS service configuration.
type NFSConfig struct {
	ID           int      `json:"id"`
	Servers      int      `json:"servers"`
	AllowNonroot bool     `json:"allow_nonroot"`
	Protocols    []string `json:"protocols"`
	V4Krb        bool     `json:"v4_krb"`
	V4Domain     string   `json:"v4_domain"`
	BindIP       []string `json:"bindip"`
	MountdPort   *int     `json:"mountd_port"`
	RpcstatdPort *int     `json:"rpcstatd_port"`
	RpclockdPort *int     `json:"rpclockd_port"`
}

// NFSConfigUpdateRequest represents the request to update NFS configuration.
type NFSConfigUpdateRequest struct {
	Servers      *int     `json:"servers,omitempty"`
	AllowNonroot *bool    `json:"allow_nonroot,omitempty"`
	Protocols    []string `json:"protocols,omitempty"`
	V4Krb        *bool    `json:"v4_krb,omitempty"`
	V4Domain     *string  `json:"v4_domain,omitempty"`
	BindIP       []string `json:"bindip,omitempty"`
	MountdPort   *int     `json:"mountd_port,omitempty"`
	RpcstatdPort *int     `json:"rpcstatd_port,omitempty"`
	RpclockdPort *int     `json:"rpclockd_port,omitempty"`
}
