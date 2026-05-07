package types

import "encoding/json"

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
