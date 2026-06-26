package types

// Request/response shapes for the filesystem.{mkdir, stat, setperm}
// namespace, used by the truenas_directory resource. Tag and pointer
// conventions match filesystem_acl.go: snake_case json keys, pointers
// with omitempty for optional fields.

// MkdirRequest is the body for filesystem.mkdir. It is passed as one
// positional object arg: filesystem.mkdir({path, options}).
type MkdirRequest struct {
	Path    string        `json:"path"`
	Options *MkdirOptions `json:"options,omitempty"`
}

// MkdirOptions configures filesystem.mkdir.
type MkdirOptions struct {
	// Mode is the octal permission string, e.g. "755".
	Mode string `json:"mode,omitempty"`
	// RaiseChmodError, when false, suppresses the error if the post-mkdir
	// chmod cannot fully apply the requested mode.
	RaiseChmodError *bool `json:"raise_chmod_error,omitempty"`
}

// FilesystemStat is the result of filesystem.stat (and the result of
// filesystem.mkdir, which returns the stat dict of the created dir).
type FilesystemStat struct {
	Realpath string  `json:"realpath"`
	Type     string  `json:"type"` // "DIRECTORY" | "FILE" | "SYMLINK" | "OTHER"
	Size     int64   `json:"size"`
	Mode     int     `json:"mode"` // full st_mode incl type bits; mask &0o7777 for perms
	UID      int     `json:"uid"`
	GID      int     `json:"gid"`
	ATime    float64 `json:"atime"`
	MTime    float64 `json:"mtime"`
	CTime    float64 `json:"ctime"`
	Nlink    int     `json:"nlink"`
	// ACL true means an extended ACL is present beyond the mode bits.
	ACL          bool    `json:"acl"`
	IsMountpoint bool    `json:"is_mountpoint"`
	User         *string `json:"user"`
	Group        *string `json:"group"`
}

// SetPermRequest is the body for filesystem.setperm (a job). Passed as
// one positional object arg.
type SetPermRequest struct {
	Path    string          `json:"path"`
	Mode    *string         `json:"mode,omitempty"` // octal string, e.g. "755"
	UID     *int            `json:"uid,omitempty"`
	GID     *int            `json:"gid,omitempty"`
	Options *SetPermOptions `json:"options,omitempty"`
}

// SetPermOptions configures recursion/traversal for setperm.
type SetPermOptions struct {
	StripACL  *bool `json:"stripacl,omitempty"`
	Recursive *bool `json:"recursive,omitempty"`
	Traverse  *bool `json:"traverse,omitempty"`
}
