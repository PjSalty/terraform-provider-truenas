package types

import "fmt"

// Dataset represents a ZFS dataset in TrueNAS.
type Dataset struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Pool          string `json:"pool"`
	Type          string `json:"type"`
	Compression   string `json:"compression,omitempty"`
	Atime         string `json:"atime,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Quota         int64  `json:"quota,omitempty"`
	Refquota      int64  `json:"refquota,omitempty"`
	Comments      string `json:"comments,omitempty"`
	MountPoint    string `json:"mountpoint,omitempty"`
	Sync          string `json:"sync,omitempty"`
	Snapdir       string `json:"snapdir,omitempty"`
	Copies        int    `json:"copies,omitempty"`
	Readonly      string `json:"readonly,omitempty"`
	RecordSize    string `json:"recordsize,omitempty"`
	Casesensitive string `json:"casesensitivity,omitempty"`
	ShareType     string `json:"share_type,omitempty"`
}

// DatasetCreateRequest represents the request body for creating a dataset.
type DatasetCreateRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type,omitempty"`
	Compression   string `json:"compression,omitempty"`
	Atime         string `json:"atime,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Quota         int64  `json:"quota,omitempty"`
	Refquota      int64  `json:"refquota,omitempty"`
	Comments      string `json:"comments,omitempty"`
	Sync          string `json:"sync,omitempty"`
	Snapdir       string `json:"snapdir,omitempty"`
	Copies        int    `json:"copies,omitempty"`
	Readonly      string `json:"readonly,omitempty"`
	RecordSize    string `json:"recordsize,omitempty"`
	ShareType     string `json:"share_type,omitempty"`
}

// DatasetUpdateRequest represents the request body for updating a dataset.
type DatasetUpdateRequest struct {
	Compression   string `json:"compression,omitempty"`
	Atime         string `json:"atime,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Quota         int64  `json:"quota,omitempty"`
	Refquota      int64  `json:"refquota,omitempty"`
	Comments      string `json:"comments,omitempty"`
	Sync          string `json:"sync,omitempty"`
	Snapdir       string `json:"snapdir,omitempty"`
	Copies        int    `json:"copies,omitempty"`
	Readonly      string `json:"readonly,omitempty"`
	RecordSize    string `json:"recordsize,omitempty"`
}

// DatasetResponse represents the API response for a dataset, where properties are objects.
//
// SCALE version notes:
//   - 25.04 and earlier: top-level `comments` is `*PropertyValue`
//   - 25.10+: top-level `comments` is always null; comments now live under
//     `user_properties.comments.parsed` — see UserProperties below.
type DatasetResponse struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Pool           string                    `json:"pool"`
	Type           string                    `json:"type"`
	MountPoint     string                    `json:"mountpoint"`
	Comments       *PropertyValue            `json:"comments"`
	UserProperties map[string]*PropertyValue `json:"user_properties"`
	Quota          *PropertyRawVal           `json:"quota"`
	Refquota       *PropertyRawVal           `json:"refquota"`
	Compression    *PropertyValue            `json:"compression"`
	Atime          *PropertyValue            `json:"atime"`
	Deduplication  *PropertyValue            `json:"deduplication"`
	Sync           *PropertyValue            `json:"sync"`
	Snapdir        *PropertyValue            `json:"snapdir"`
	Copies         *PropertyValue            `json:"copies"`
	Readonly       *PropertyValue            `json:"readonly"`
	RecordSize     *PropertyValue            `json:"recordsize"`
	ShareType      *PropertyValue            `json:"share_type"`
	// Zvol-specific properties — only populated when Type == "VOLUME".
	Volsize      *PropertyRawVal `json:"volsize"`
	Volblocksize *PropertyValue  `json:"volblocksize"`
}

// GetVolsize returns the raw zvol size in bytes. Returns 0 for non-zvol
// datasets or when the property is not set.
func (d *DatasetResponse) GetVolsize() int64 {
	if d.Volsize == nil || d.Volsize.Rawvalue == "" {
		return 0
	}
	// Rawvalue is a decimal string like "16777216".
	var n int64
	if _, err := fmt.Sscanf(d.Volsize.Rawvalue, "%d", &n); err == nil {
		return n
	}
	return 0
}

// GetVolblocksize returns the zvol block size (e.g. "16K"). Returns empty
// string for non-zvol datasets or when the property is not set.
func (d *DatasetResponse) GetVolblocksize() string {
	if d.Volblocksize == nil {
		return ""
	}
	return d.Volblocksize.Value
}

// GetComments returns the dataset's comments string across SCALE versions:
//   - 25.10+: user_properties["comments"].parsed / .value
//   - 25.04 and earlier: top-level Comments.Value
//
// Returns empty string if not set in either location.
func (d *DatasetResponse) GetComments() string {
	if d.UserProperties != nil {
		if pv, ok := d.UserProperties["comments"]; ok && pv != nil {
			if pv.Value != "" {
				return pv.Value
			}
		}
	}
	if d.Comments != nil && d.Comments.Value != "" {
		return d.Comments.Value
	}
	return ""
}
