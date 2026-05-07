package types

// Group represents a local group in TrueNAS.
type Group struct {
	ID             int      `json:"id"`
	GID            int      `json:"gid"`
	Name           string   `json:"name"`
	Builtin        bool     `json:"builtin"`
	SMB            bool     `json:"smb"`
	SudoCommands   []string `json:"sudo_commands"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd"`
	Users          []int    `json:"users"`
}

// GroupCreateRequest represents the request to create a group.
type GroupCreateRequest struct {
	Name           string   `json:"name"`
	GID            int      `json:"gid,omitempty"`
	SMB            bool     `json:"smb"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}

// GroupUpdateRequest represents the request to update a group.
type GroupUpdateRequest struct {
	Name           string   `json:"name,omitempty"`
	SMB            *bool    `json:"smb,omitempty"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}
