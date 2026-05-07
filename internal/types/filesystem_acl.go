package types

// FilesystemACL represents the ACL on a filesystem path.
type FilesystemACL struct {
	Path    string     `json:"path"`
	UID     int        `json:"uid"`
	GID     int        `json:"gid"`
	ACLType string     `json:"acltype"`
	ACL     []ACLEntry `json:"acl"`
	Trivial bool       `json:"trivial"`
	User    *string    `json:"user"`
	Group   *string    `json:"group"`
}

// ACLEntry represents a single ACL entry.
type ACLEntry struct {
	Tag     string   `json:"tag"`
	ID      int      `json:"id"`
	Perms   ACLPerms `json:"perms"`
	Default bool     `json:"default"`
	Who     *string  `json:"who"`
}

// ACLPerms represents POSIX ACL permissions.
type ACLPerms struct {
	Read    bool `json:"READ"`
	Write   bool `json:"WRITE"`
	Execute bool `json:"EXECUTE"`
}

// SetACLRequest is the body for POST /filesystem/setacl /
// filesystem.setacl.
type SetACLRequest struct {
	Path    string        `json:"path"`
	DACL    []SetACLEntry `json:"dacl"`
	ACLType string        `json:"acltype,omitempty"`
	UID     *int          `json:"uid,omitempty"`
	GID     *int          `json:"gid,omitempty"`
}

// SetACLEntry represents a single ACL entry in a set request.
type SetACLEntry struct {
	Tag     string   `json:"tag"`
	ID      int      `json:"id"`
	Perms   ACLPerms `json:"perms"`
	Default bool     `json:"default"`
}

// GetACLRequest is the body for POST /filesystem/getacl /
// filesystem.getacl.
type GetACLRequest struct {
	Path string `json:"path"`
}
