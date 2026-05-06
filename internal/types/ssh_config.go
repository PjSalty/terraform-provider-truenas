package types

// SSHConfig represents the SSH service configuration.
type SSHConfig struct {
	ID              int      `json:"id"`
	TCPPort         int      `json:"tcpport"`
	PasswordAuth    bool     `json:"passwordauth"`
	KerberosAuth    bool     `json:"kerberosauth"`
	TCPFwd          bool     `json:"tcpfwd"`
	Compression     bool     `json:"compression"`
	SFTPLogLevel    string   `json:"sftp_log_level"`
	SFTPLogFacility string   `json:"sftp_log_facility"`
	WeakCiphers     []string `json:"weak_ciphers"`
}

// SSHConfigUpdateRequest represents the request to update SSH configuration.
type SSHConfigUpdateRequest struct {
	TCPPort         *int      `json:"tcpport,omitempty"`
	PasswordAuth    *bool     `json:"passwordauth,omitempty"`
	KerberosAuth    *bool     `json:"kerberosauth,omitempty"`
	TCPFwd          *bool     `json:"tcpfwd,omitempty"`
	Compression     *bool     `json:"compression,omitempty"`
	SFTPLogLevel    *string   `json:"sftp_log_level,omitempty"`
	SFTPLogFacility *string   `json:"sftp_log_facility,omitempty"`
	WeakCiphers     *[]string `json:"weak_ciphers,omitempty"`
}
