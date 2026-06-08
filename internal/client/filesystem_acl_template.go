package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// FilesystemACLTemplate represents a named ACL template.
// ACL entries are polymorphic (NFS4/POSIX1E) so we keep them as raw JSON.
type FilesystemACLTemplate struct {
	ID      int             `json:"id"`
	Name    string          `json:"name"`
	ACLType string          `json:"acltype"`
	Comment string          `json:"comment"`
	ACL     json.RawMessage `json:"acl,omitempty"`
	Builtin bool            `json:"builtin"`
}

// FilesystemACLTemplateCreateRequest is the create payload.
type FilesystemACLTemplateCreateRequest struct {
	Name    string          `json:"name"`
	ACLType string          `json:"acltype"`
	Comment string          `json:"comment,omitempty"`
	ACL     json.RawMessage `json:"acl"`
}

// FilesystemACLTemplateUpdateRequest is the update payload.
type FilesystemACLTemplateUpdateRequest struct {
	Name    *string         `json:"name,omitempty"`
	Comment *string         `json:"comment,omitempty"`
	ACL     json.RawMessage `json:"acl,omitempty"`
}

// GetFilesystemACLTemplate retrieves an ACL template by ID.
func (c *Client) GetFilesystemACLTemplate(ctx context.Context, id int) (*FilesystemACLTemplate, error) {
	tflog.Trace(ctx, "GetFilesystemACLTemplate start")

	resp, err := c.Get(ctx, fmt.Sprintf("/filesystem/acltemplate/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting filesystem ACL template %d: %w", id, err)
	}

	var t FilesystemACLTemplate
	if err := json.Unmarshal(resp, &t); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL template response: %w", err)
	}

	tflog.Trace(ctx, "GetFilesystemACLTemplate success")
	return &t, nil
}

// CreateFilesystemACLTemplate creates an ACL template.
func (c *Client) CreateFilesystemACLTemplate(ctx context.Context, req *FilesystemACLTemplateCreateRequest) (*FilesystemACLTemplate, error) {
	tflog.Trace(ctx, "CreateFilesystemACLTemplate start")

	resp, err := c.Post(ctx, "/filesystem/acltemplate", req)
	if err != nil {
		return nil, fmt.Errorf("creating filesystem ACL template: %w", err)
	}

	var t FilesystemACLTemplate
	if err := json.Unmarshal(resp, &t); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL template create response: %w", err)
	}

	tflog.Trace(ctx, "CreateFilesystemACLTemplate success")
	return &t, nil
}

// UpdateFilesystemACLTemplate updates an ACL template by ID.
func (c *Client) UpdateFilesystemACLTemplate(ctx context.Context, id int, req *FilesystemACLTemplateUpdateRequest) (*FilesystemACLTemplate, error) {
	tflog.Trace(ctx, "UpdateFilesystemACLTemplate start")

	resp, err := c.Put(ctx, fmt.Sprintf("/filesystem/acltemplate/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating filesystem ACL template %d: %w", id, err)
	}

	var t FilesystemACLTemplate
	if err := json.Unmarshal(resp, &t); err != nil {
		return nil, fmt.Errorf("parsing filesystem ACL template update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFilesystemACLTemplate success")
	return &t, nil
}

// DeleteFilesystemACLTemplate deletes an ACL template by ID.
func (c *Client) DeleteFilesystemACLTemplate(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteFilesystemACLTemplate start")

	_, err := c.Delete(ctx, fmt.Sprintf("/filesystem/acltemplate/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting filesystem ACL template %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteFilesystemACLTemplate success")
	return nil
}
