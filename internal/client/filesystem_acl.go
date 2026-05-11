package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Filesystem ACL API ---

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

// SetACLRequest represents the request to set an ACL on a path.
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

// GetACLRequest represents the request to get ACL for a path.
type GetACLRequest struct {
	Path string `json:"path"`
}

// GetFilesystemACL retrieves the ACL for a path.
func (c *Client) GetFilesystemACL(ctx context.Context, path string) (*FilesystemACL, error) {
	tflog.Trace(ctx, "GetFilesystemACL start")

	resp, err := c.Post(ctx, "/filesystem/getacl", &GetACLRequest{Path: path})
	if err != nil {
		return nil, fmt.Errorf("getting filesystem ACL for %q: %w", path, err)
	}

	var acl FilesystemACL
	if err := json.Unmarshal(resp, &acl); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL response: %w", err)
	}

	tflog.Trace(ctx, "GetFilesystemACL success")
	return &acl, nil
}

// SetFilesystemACL sets the ACL on a path. Returns a job ID.
func (c *Client) SetFilesystemACL(ctx context.Context, req *SetACLRequest) error {
	tflog.Trace(ctx, "SetFilesystemACL start")

	_, err := c.Post(ctx, "/filesystem/setacl", req)
	if err != nil {
		return fmt.Errorf("setting filesystem ACL for %q: %w", req.Path, err)
	}

	tflog.Trace(ctx, "SetFilesystemACL success")
	return nil
}
