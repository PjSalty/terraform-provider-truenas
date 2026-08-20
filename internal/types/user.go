package types

import "encoding/json"

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
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email,omitempty"`
	// Password is a pointer so a passwordless account can omit the key
	// entirely. Middleware's __set_password returns early when the key is
	// absent, and UserCreate defaults password to None, which leaves
	// unixhash as '*'. Sending "" instead would be rejected by the
	// NonEmptyString model.
	Password         *string  `json:"password,omitempty"`
	PasswordDisabled *bool    `json:"password_disabled,omitempty"`
	UID              int      `json:"uid,omitempty"`
	Group            int      `json:"group,omitempty"`
	GroupCreate      bool     `json:"group_create"`
	Groups           []int    `json:"groups,omitempty"`
	Home             string   `json:"home,omitempty"`
	Shell            string   `json:"shell,omitempty"`
	Locked           bool     `json:"locked"`
	SMB              bool     `json:"smb"`
	SSHPubKey        string   `json:"sshpubkey,omitempty"`
	SudoCommands     []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP   []string `json:"sudo_commands_nopasswd,omitempty"`
}

// UserUpdateRequest represents the request to update a user.
type UserUpdateRequest struct {
	FullName string `json:"full_name,omitempty"`
	Email    string `json:"email,omitempty"`
	// Password needs THREE wire states on update, which *string cannot
	// express:
	//
	//   omitted      leave the stored hash alone (manage an account
	//                without knowing or rotating its password)
	//   null         wipe it: __set_password treats a falsy value as
	//                unixhash='*', which is how an account becomes
	//                passwordless
	//   "value"      set it
	//
	// nil RawMessage omits the key, RawMessage("null") emits an explicit
	// null, and a marshaled string emits the value. Use SetPassword and
	// ClearPassword rather than assigning this directly.
	Password         json.RawMessage `json:"password,omitempty"`
	PasswordDisabled *bool           `json:"password_disabled,omitempty"`
	Group            int             `json:"group,omitempty"`
	Groups           []int           `json:"groups,omitempty"`
	Home             string          `json:"home,omitempty"`
	Shell            string          `json:"shell,omitempty"`
	Locked           *bool           `json:"locked,omitempty"`
	SMB              *bool           `json:"smb,omitempty"`
	SSHPubKey        string          `json:"sshpubkey,omitempty"`
	SudoCommands     []string        `json:"sudo_commands,omitempty"`
	SudoCommandsNP   []string        `json:"sudo_commands_nopasswd,omitempty"`
}

// SetPassword puts an explicit password on the wire.
//
// No error return: json.Marshal of a Go string cannot fail. Invalid UTF-8 is
// replaced with U+FFFD rather than rejected, and there is no other failure
// mode for a string. An error return here would be a branch no test could
// ever reach.
func (r *UserUpdateRequest) SetPassword(pw string) {
	b, _ := json.Marshal(pw)
	r.Password = b
}

// ClearPassword emits an explicit JSON null, which middleware reads as
// "wipe the hash" rather than "leave it alone". Omitting the key instead
// would silently keep the old password on a supposedly passwordless
// account.
func (r *UserUpdateRequest) ClearPassword() {
	r.Password = json.RawMessage("null")
}
