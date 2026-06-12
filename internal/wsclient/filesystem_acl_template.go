package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for ACL templates:
// filesystem.acltemplate.{get_instance, create, update, delete}.

// GetFilesystemACLTemplate retrieves an ACL template by ID.
func (c *Client) GetFilesystemACLTemplate(ctx context.Context, id int) (*types.FilesystemACLTemplate, error) {
	tflog.Trace(ctx, "GetFilesystemACLTemplate (ws) start")

	result, err := c.Call(ctx, "filesystem.acltemplate.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting filesystem ACL template %d: %w", id, err)
	}

	var tmpl types.FilesystemACLTemplate
	if err := json.Unmarshal(result, &tmpl); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL template response: %w", err)
	}

	tflog.Trace(ctx, "GetFilesystemACLTemplate (ws) success")
	return &tmpl, nil
}

// CreateFilesystemACLTemplate creates an ACL template.
func (c *Client) CreateFilesystemACLTemplate(ctx context.Context, req *types.FilesystemACLTemplateCreateRequest) (*types.FilesystemACLTemplate, error) {
	tflog.Trace(ctx, "CreateFilesystemACLTemplate (ws) start")

	result, err := c.Call(ctx, "filesystem.acltemplate.create",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("creating filesystem ACL template: %w", err)
	}

	var tmpl types.FilesystemACLTemplate
	if err := json.Unmarshal(result, &tmpl); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL template create response: %w", err)
	}

	tflog.Trace(ctx, "CreateFilesystemACLTemplate (ws) success")
	return &tmpl, nil
}

// UpdateFilesystemACLTemplate updates an ACL template by ID.
func (c *Client) UpdateFilesystemACLTemplate(ctx context.Context, id int, req *types.FilesystemACLTemplateUpdateRequest) (*types.FilesystemACLTemplate, error) {
	tflog.Trace(ctx, "UpdateFilesystemACLTemplate (ws) start")

	result, err := c.Call(ctx, "filesystem.acltemplate.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating filesystem ACL template %d: %w", id, err)
	}

	var tmpl types.FilesystemACLTemplate
	if err := json.Unmarshal(result, &tmpl); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL template update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFilesystemACLTemplate (ws) success")
	return &tmpl, nil
}

// DeleteFilesystemACLTemplate deletes an ACL template by ID.
func (c *Client) DeleteFilesystemACLTemplate(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteFilesystemACLTemplate (ws) start")

	if _, err := c.Call(ctx, "filesystem.acltemplate.delete",
		[]interface{}{id},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("deleting filesystem ACL template %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteFilesystemACLTemplate (ws) success")
	return nil
}
