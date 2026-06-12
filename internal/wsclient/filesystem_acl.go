package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for filesystem ACL operations:
// filesystem.{getacl, setacl}.

// GetFilesystemACL retrieves the ACL for a path.
//
// SCALE 25.10 changed filesystem.getacl from object-args to positional
// args. The server now expects [path, simplified, resolve_ids] rather
// than [{path: ...}]. We hand-build the params array here and keep the
// defaults aligned with the upstream schema (simplified=true,
// resolve_ids=false).
func (c *Client) GetFilesystemACL(ctx context.Context, path string) (*types.FilesystemACL, error) {
	tflog.Trace(ctx, "GetFilesystemACL (ws) start")

	result, err := c.Call(ctx, "filesystem.getacl",
		[]interface{}{path, true, false},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting filesystem ACL for %q: %w", path, err)
	}

	var acl types.FilesystemACL
	if err := json.Unmarshal(result, &acl); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL response: %w", err)
	}

	tflog.Trace(ctx, "GetFilesystemACL (ws) success")
	return &acl, nil
}

// SetFilesystemACL sets the ACL on a path.
//
// Note: filesystem.setacl is technically a job server-side, but the
// REST client's SetFilesystemACL doesn't poll for completion (it
// returns as soon as the POST returns). The wsclient mirrors that
// fire-and-return behavior for behavioral parity. If the resource
// later wants strict completion semantics, switch this to CallJob.
func (c *Client) SetFilesystemACL(ctx context.Context, req *types.SetACLRequest) error {
	tflog.Trace(ctx, "SetFilesystemACL (ws) start")

	// SCALE 25.10 made filesystem.setacl a long-running job.
	// CallJob polls core.get_jobs until the job reaches terminal
	// state. Idempotent=true because re-applying the same ACL on the
	// same path is a no-op server-side.
	if _, err := c.CallJob(ctx, "filesystem.setacl",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: true},
		500*time.Millisecond); err != nil {
		return fmt.Errorf("setting filesystem ACL for %q: %w", req.Path, err)
	}

	tflog.Trace(ctx, "SetFilesystemACL (ws) success")
	return nil
}
