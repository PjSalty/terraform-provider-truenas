package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- Filesystem ACL API ---

// FilesystemACL, ACLEntry, ACLPerms, SetACLRequest, SetACLEntry,
// GetACLRequest moved to internal/types/filesystem_acl.go in the v2.0
// transport-migration prep.
type (
	FilesystemACL = types.FilesystemACL
	ACLEntry      = types.ACLEntry
	ACLPerms      = types.ACLPerms
	SetACLRequest = types.SetACLRequest
	SetACLEntry   = types.SetACLEntry
	GetACLRequest = types.GetACLRequest
)

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
