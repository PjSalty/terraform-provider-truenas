package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method names for init/shutdown scripts. The TrueNAS WebSocket
// API name is "initshutdownscript", matching the REST base path.

// GetInitScript retrieves an init/startup script by ID.
func (c *Client) GetInitScript(ctx context.Context, id int) (*types.InitScript, error) {
	tflog.Trace(ctx, "GetInitScript (ws) start")

	result, err := c.Call(ctx, "initshutdownscript.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting init script %d: %w", id, err)
	}

	var s types.InitScript
	if err := json.Unmarshal(result, &s); err != nil {
		return nil, fmt.Errorf("parsing init script response: %w", err)
	}

	tflog.Trace(ctx, "GetInitScript (ws) success")
	return &s, nil
}

// CreateInitScript creates a new init/startup script.
func (c *Client) CreateInitScript(ctx context.Context, req *types.InitScriptCreateRequest) (*types.InitScript, error) {
	tflog.Trace(ctx, "CreateInitScript (ws) start")

	result, err := c.Call(ctx, "initshutdownscript.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating init script: %w", err)
	}

	var s types.InitScript
	if err := json.Unmarshal(result, &s); err != nil {
		return nil, fmt.Errorf("parsing init script create response: %w", err)
	}

	tflog.Trace(ctx, "CreateInitScript (ws) success")
	return &s, nil
}

// UpdateInitScript updates an existing init/startup script. The JSON-RPC
// params array carries [id, partial-config].
func (c *Client) UpdateInitScript(ctx context.Context, id int, req *types.InitScriptUpdateRequest) (*types.InitScript, error) {
	tflog.Trace(ctx, "UpdateInitScript (ws) start")

	result, err := c.Call(ctx, "initshutdownscript.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating init script %d: %w", id, err)
	}

	var s types.InitScript
	if err := json.Unmarshal(result, &s); err != nil {
		return nil, fmt.Errorf("parsing init script update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateInitScript (ws) success")
	return &s, nil
}

// DeleteInitScript deletes an init/startup script by ID.
func (c *Client) DeleteInitScript(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteInitScript (ws) start")

	if _, err := c.Call(ctx, "initshutdownscript.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting init script %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteInitScript (ws) success")
	return nil
}
