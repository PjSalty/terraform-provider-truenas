package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- Init/Startup Script API ---

// InitScript, InitScriptCreateRequest, InitScriptUpdateRequest moved to
// internal/types/init_script.go in the v2.0 transport-migration prep.
// Aliased here so existing imports keep compiling.
type (
	InitScript              = types.InitScript
	InitScriptCreateRequest = types.InitScriptCreateRequest
	InitScriptUpdateRequest = types.InitScriptUpdateRequest
)

// initScriptBasePath is the API path for init/startup scripts.
const initScriptBasePath = "/initshutdownscript"

// GetInitScript retrieves an init/startup script by ID.
func (c *Client) GetInitScript(ctx context.Context, id int) (*InitScript, error) {
	tflog.Trace(ctx, "GetInitScript start")

	resp, err := c.Get(ctx, fmt.Sprintf("%s/id/%d", initScriptBasePath, id))
	if err != nil {
		return nil, fmt.Errorf("getting init script %d: %w", id, err)
	}

	var script InitScript
	if err := json.Unmarshal(resp, &script); err != nil {
		return nil, fmt.Errorf("parsing init script response: %w", err)
	}

	tflog.Trace(ctx, "GetInitScript success")
	return &script, nil
}

// CreateInitScript creates a new init/startup script.
func (c *Client) CreateInitScript(ctx context.Context, req *InitScriptCreateRequest) (*InitScript, error) {
	tflog.Trace(ctx, "CreateInitScript start")

	resp, err := c.Post(ctx, initScriptBasePath, req)
	if err != nil {
		return nil, fmt.Errorf("creating init script: %w", err)
	}

	var script InitScript
	if err := json.Unmarshal(resp, &script); err != nil {
		return nil, fmt.Errorf("parsing init script create response: %w", err)
	}

	tflog.Trace(ctx, "CreateInitScript success")
	return &script, nil
}

// UpdateInitScript updates an existing init/startup script.
func (c *Client) UpdateInitScript(ctx context.Context, id int, req *InitScriptUpdateRequest) (*InitScript, error) {
	tflog.Trace(ctx, "UpdateInitScript start")

	resp, err := c.Put(ctx, fmt.Sprintf("%s/id/%d", initScriptBasePath, id), req)
	if err != nil {
		return nil, fmt.Errorf("updating init script %d: %w", id, err)
	}

	var script InitScript
	if err := json.Unmarshal(resp, &script); err != nil {
		return nil, fmt.Errorf("parsing init script update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateInitScript success")
	return &script, nil
}

// DeleteInitScript deletes an init/startup script.
func (c *Client) DeleteInitScript(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteInitScript start")

	_, err := c.Delete(ctx, fmt.Sprintf("%s/id/%d", initScriptBasePath, id))
	if err != nil {
		return fmt.Errorf("deleting init script %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteInitScript success")
	return nil
}
