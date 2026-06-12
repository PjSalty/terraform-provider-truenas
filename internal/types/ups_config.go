package types

// UPSConfig represents the UPS configuration.
type UPSConfig struct {
	ID            int    `json:"id"`
	Mode          string `json:"mode"`
	Identifier    string `json:"identifier"`
	Driver        string `json:"driver"`
	Port          string `json:"port"`
	RemoteHost    string `json:"remotehost"`
	RemotePort    int    `json:"remoteport"`
	Shutdown      string `json:"shutdown"`
	ShutdownTimer int    `json:"shutdowntimer"`
	Description   string `json:"description"`
}

// UPSConfigUpdateRequest represents the request to update UPS configuration.
type UPSConfigUpdateRequest struct {
	Mode          *string `json:"mode,omitempty"`
	Identifier    *string `json:"identifier,omitempty"`
	Driver        *string `json:"driver,omitempty"`
	Port          *string `json:"port,omitempty"`
	RemoteHost    *string `json:"remotehost,omitempty"`
	RemotePort    *int    `json:"remoteport,omitempty"`
	Shutdown      *string `json:"shutdown,omitempty"`
	ShutdownTimer *int    `json:"shutdowntimer,omitempty"`
	Description   *string `json:"description,omitempty"`
}
