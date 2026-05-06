package types

// MailConfig represents the mail/SMTP configuration.
type MailConfig struct {
	ID             int     `json:"id"`
	FromEmail      string  `json:"fromemail"`
	FromName       string  `json:"fromname"`
	OutgoingServer string  `json:"outgoingserver"`
	Port           int     `json:"port"`
	Security       string  `json:"security"`
	SMTP           bool    `json:"smtp"`
	User           *string `json:"user"`
	Pass           string  `json:"pass"`
}

// MailConfigUpdateRequest represents the request to update mail configuration.
type MailConfigUpdateRequest struct {
	FromEmail      *string `json:"fromemail,omitempty"`
	FromName       *string `json:"fromname,omitempty"`
	OutgoingServer *string `json:"outgoingserver,omitempty"`
	Port           *int    `json:"port,omitempty"`
	Security       *string `json:"security,omitempty"`
	SMTP           *bool   `json:"smtp,omitempty"`
	User           *string `json:"user,omitempty"`
	Pass           *string `json:"pass,omitempty"`
}
