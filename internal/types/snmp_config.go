package types

// SNMPConfig represents the SNMP configuration.
type SNMPConfig struct {
	ID               int     `json:"id"`
	Community        string  `json:"community"`
	Contact          string  `json:"contact"`
	Location         string  `json:"location"`
	V3               bool    `json:"v3"`
	V3Username       string  `json:"v3_username"`
	V3AuthType       string  `json:"v3_authtype"`
	V3Password       string  `json:"v3_password"`
	V3PrivProto      *string `json:"v3_privproto"`
	V3PrivPassphrase *string `json:"v3_privpassphrase"`
}

// SNMPConfigUpdateRequest represents the request to update SNMP configuration.
type SNMPConfigUpdateRequest struct {
	Community        *string `json:"community,omitempty"`
	Contact          *string `json:"contact,omitempty"`
	Location         *string `json:"location,omitempty"`
	V3               *bool   `json:"v3,omitempty"`
	V3Username       *string `json:"v3_username,omitempty"`
	V3AuthType       *string `json:"v3_authtype,omitempty"`
	V3Password       *string `json:"v3_password,omitempty"`
	V3PrivProto      *string `json:"v3_privproto,omitempty"`
	V3PrivPassphrase *string `json:"v3_privpassphrase,omitempty"`
}
