package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for filesystem path operations:
// filesystem.{mkdir, stat, setperm}. Used by the truenas_directory
// resource. mkdir and stat are synchronous Calls; setperm is a job.

// Mkdir creates a directory via filesystem.mkdir. The method is a
// synchronous Call (not a job) and returns the stat dict of the new
// directory. The request marshals to {path, options{mode,
// raise_chmod_error}} and is passed as one positional object arg.
func (c *Client) Mkdir(ctx context.Context, req *types.MkdirRequest) (*types.FilesystemStat, error) {
	tflog.Trace(ctx, "Mkdir (ws) start")

	result, err := c.Call(ctx, "filesystem.mkdir",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating directory %q: %w", req.Path, err)
	}

	var stat types.FilesystemStat
	if err := json.Unmarshal(result, &stat); err != nil {
		return nil, fmt.Errorf("parsing mkdir response: %w", err)
	}

	tflog.Trace(ctx, "Mkdir (ws) success")
	return &stat, nil
}

// StatFilesystem reads a path via filesystem.stat. Positional arg is
// [path]. Read-only and idempotent. A missing path surfaces as ENOENT,
// which wsclient.IsNotFound recognizes.
func (c *Client) StatFilesystem(ctx context.Context, path string) (*types.FilesystemStat, error) {
	tflog.Trace(ctx, "StatFilesystem (ws) start")

	result, err := c.Call(ctx, "filesystem.stat",
		[]interface{}{path},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("stating path %q: %w", path, err)
	}

	var stat types.FilesystemStat
	if err := json.Unmarshal(result, &stat); err != nil {
		return nil, fmt.Errorf("parsing stat response: %w", err)
	}

	tflog.Trace(ctx, "StatFilesystem (ws) success")
	return &stat, nil
}

// SetFilesystemPerm applies mode/uid/gid via filesystem.setperm. This
// is a job on SCALE 25.04, so we use CallJob and poll for terminal
// state, exactly like SetFilesystemACL. Idempotent because re-applying
// the same perms on the same path is a server-side no-op. The request
// is passed as one positional object arg.
func (c *Client) SetFilesystemPerm(ctx context.Context, req *types.SetPermRequest) error {
	tflog.Trace(ctx, "SetFilesystemPerm (ws) start")

	if _, err := c.CallJob(ctx, "filesystem.setperm",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: true},
		500*time.Millisecond); err != nil {
		return fmt.Errorf("setting perms on %q: %w", req.Path, err)
	}

	tflog.Trace(ctx, "SetFilesystemPerm (ws) success")
	return nil
}
