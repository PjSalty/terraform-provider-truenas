package types

// User represents a local user in TrueNAS.
type User struct {
	ID               int       `json:"id"`
	UID              int       `json:"uid"`
	Username         string    `json:"username"`
	FullName         string    `json:"full_name"`
	Email            *string   `json:"email"`
	Home             string    `json:"home"`
	Shell            string    `json:"shell"`
	Builtin          bool      `json:"builtin"`
	Locked           bool      `json:"locked"`
	SMB              bool      `json:"smb"`
	SSHPubKey        *string   `json:"sshpubkey"`
	PasswordDisabled bool      `json:"password_disabled"`
	Group            UserGroup `json:"group"`
	Groups           []int     `json:"groups"`
	SudoCommands     []string  `json:"sudo_commands"`
	SudoCommandsNP   []string  `json:"sudo_commands_nopasswd"`
}

// UserGroup represents the primary group of a user.
type UserGroup struct {
	ID    int    `json:"id"`
	GID   int    `json:"bsdgrp_gid"`
	Group string `json:"bsdgrp_group"`
}

// UserCreateRequest represents the request to create a user.
type UserCreateRequest struct {
	Username       string   `json:"username"`
	FullName       string   `json:"full_name"`
	Email          string   `json:"email,omitempty"`
	Password       string   `json:"password"`
	UID            int      `json:"uid,omitempty"`
	Group          int      `json:"group,omitempty"`
	GroupCreate    bool     `json:"group_create"`
	Groups         []int    `json:"groups,omitempty"`
	Home           string   `json:"home,omitempty"`
	Shell          string   `json:"shell,omitempty"`
	Locked         bool     `json:"locked"`
	SMB            bool     `json:"smb"`
	SSHPubKey      string   `json:"sshpubkey,omitempty"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}

// UserUpdateRequest represents the request to update a user.
type UserUpdateRequest struct {
	FullName       string   `json:"full_name,omitempty"`
	Email          string   `json:"email,omitempty"`
	Password       string   `json:"password,omitempty"`
	Group          int      `json:"group,omitempty"`
	Groups         []int    `json:"groups,omitempty"`
	Home           string   `json:"home,omitempty"`
	Shell          string   `json:"shell,omitempty"`
	Locked         *bool    `json:"locked,omitempty"`
	SMB            *bool    `json:"smb,omitempty"`
	SSHPubKey      string   `json:"sshpubkey,omitempty"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}
