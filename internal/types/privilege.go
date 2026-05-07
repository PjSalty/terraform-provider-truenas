package types

import "fmt"

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
