package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Privilege represents a TrueNAS privilege (RBAC grant).
//
// The API returns local_groups and ds_groups as enriched objects (with
// id/gid/name), but create/update accept simple lists of GIDs (for
// local_groups) and GIDs or SID strings (for ds_groups). We capture the full
// GET response via PrivilegeGroup and expose helpers to extract GIDs.
type Privilege struct {
	ID          int              `json:"id"`
	BuiltinName *string          `json:"builtin_name,omitempty"`
	Name        string           `json:"name"`
	LocalGroups []PrivilegeGroup `json:"local_groups"`
	DSGroups    []interface{}    `json:"ds_groups"`
	Roles       []string         `json:"roles"`
	WebShell    bool             `json:"web_shell"`
}

// PrivilegeGroup is the enriched group object returned by privilege.query.
type PrivilegeGroup struct {
	ID   int    `json:"id"`
	GID  int    `json:"gid"`
	Name string `json:"name"`
}

// LocalGroupGIDs returns just the GIDs from the enriched local_groups list.
func (p *Privilege) LocalGroupGIDs() []int {
	out := make([]int, 0, len(p.LocalGroups))
	for _, g := range p.LocalGroups {
		out = append(out, g.GID)
	}
	return out
}

// DSGroupStrings returns ds_groups as a string slice. Entries may be
// integers (GIDs) or strings (SIDs). We stringify everything for storage.
func (p *Privilege) DSGroupStrings() []string {
	out := make([]string, 0, len(p.DSGroups))
	for _, g := range p.DSGroups {
		switch v := g.(type) {
		case string:
			out = append(out, v)
		case float64:
			out = append(out, fmt.Sprintf("%d", int(v)))
		case int:
			out = append(out, fmt.Sprintf("%d", v))
		}
	}
	return out
}

// PrivilegeCreateRequest is the body for POST /privilege.
type PrivilegeCreateRequest struct {
	Name        string        `json:"name"`
	LocalGroups []int         `json:"local_groups"`
	DSGroups    []interface{} `json:"ds_groups"`
	Roles       []string      `json:"roles"`
	WebShell    bool          `json:"web_shell"`
}

// PrivilegeUpdateRequest is the body for PUT /privilege/id/{id}.
type PrivilegeUpdateRequest struct {
	Name        *string        `json:"name,omitempty"`
	LocalGroups *[]int         `json:"local_groups,omitempty"`
	DSGroups    *[]interface{} `json:"ds_groups,omitempty"`
	Roles       *[]string      `json:"roles,omitempty"`
	WebShell    *bool          `json:"web_shell,omitempty"`
}

// ListPrivileges retrieves all privileges.
func (c *Client) ListPrivileges(ctx context.Context) ([]Privilege, error) {
	tflog.Trace(ctx, "ListPrivileges start")

	resp, err := c.Get(ctx, "/privilege")
	if err != nil {
		return nil, fmt.Errorf("listing privileges: %w", err)
	}

	var items []Privilege
	if err := json.Unmarshal(resp, &items); err != nil {
		return nil, fmt.Errorf("parsing privileges list response: %w", err)
	}
	tflog.Trace(ctx, "ListPrivileges success")
	return items, nil
}

// GetPrivilege retrieves a privilege by ID.
func (c *Client) GetPrivilege(ctx context.Context, id int) (*Privilege, error) {
	tflog.Trace(ctx, "GetPrivilege start")

	resp, err := c.Get(ctx, fmt.Sprintf("/privilege/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting privilege %d: %w", id, err)
	}

	var p Privilege
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege response: %w", err)
	}
	tflog.Trace(ctx, "GetPrivilege success")
	return &p, nil
}

// CreatePrivilege creates a new privilege.
func (c *Client) CreatePrivilege(ctx context.Context, req *PrivilegeCreateRequest) (*Privilege, error) {
	tflog.Trace(ctx, "CreatePrivilege start")

	resp, err := c.Post(ctx, "/privilege", req)
	if err != nil {
		return nil, fmt.Errorf("creating privilege: %w", err)
	}

	var p Privilege
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege create response: %w", err)
	}
	tflog.Trace(ctx, "CreatePrivilege success")
	return &p, nil
}

// UpdatePrivilege updates an existing privilege.
func (c *Client) UpdatePrivilege(ctx context.Context, id int, req *PrivilegeUpdateRequest) (*Privilege, error) {
	tflog.Trace(ctx, "UpdatePrivilege start")

	resp, err := c.Put(ctx, fmt.Sprintf("/privilege/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating privilege %d: %w", id, err)
	}

	var p Privilege
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing privilege update response: %w", err)
	}
	tflog.Trace(ctx, "UpdatePrivilege success")
	return &p, nil
}

// DeletePrivilege deletes a privilege.
func (c *Client) DeletePrivilege(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeletePrivilege start")

	_, err := c.Delete(ctx, fmt.Sprintf("/privilege/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting privilege %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeletePrivilege success")
	return nil
}
