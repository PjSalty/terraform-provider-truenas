package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Dataset API ---

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

// GetDataset retrieves a dataset by its ID (full path like "tank/dataset").
func (c *Client) GetDataset(ctx context.Context, id string) (*DatasetResponse, error) {
	tflog.Trace(ctx, "GetDataset start")

	encodedID := url.PathEscape(id)
	resp, err := c.Get(ctx, "/pool/dataset/id/"+encodedID)
	if err != nil {
		return nil, fmt.Errorf("getting dataset %q: %w", id, err)
	}

	var dataset DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset response: %w", err)
	}

	tflog.Trace(ctx, "GetDataset success")
	return &dataset, nil
}

// ListDatasets retrieves all datasets. The optional pool filter limits
// results to datasets belonging to the named pool; the optional parent
// filter limits results to datasets whose ID has the given prefix followed
// by a '/'. Filtering is performed client-side.
func (c *Client) ListDatasets(ctx context.Context) ([]DatasetResponse, error) {
	tflog.Trace(ctx, "ListDatasets start")

	resp, err := c.Get(ctx, "/pool/dataset")
	if err != nil {
		return nil, fmt.Errorf("listing datasets: %w", err)
	}

	var datasets []DatasetResponse
	if err := json.Unmarshal(resp, &datasets); err != nil {
		return nil, fmt.Errorf("parsing datasets list response: %w", err)
	}
	tflog.Trace(ctx, "ListDatasets success")
	return datasets, nil
}

// CreateDataset creates a new ZFS dataset.
func (c *Client) CreateDataset(ctx context.Context, req *DatasetCreateRequest) (*DatasetResponse, error) {
	tflog.Trace(ctx, "CreateDataset start")

	resp, err := c.Post(ctx, "/pool/dataset", req)
	if err != nil {
		return nil, fmt.Errorf("creating dataset %q: %w", req.Name, err)
	}

	var dataset DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset create response: %w", err)
	}

	tflog.Trace(ctx, "CreateDataset success")
	return &dataset, nil
}

// UpdateDataset updates an existing ZFS dataset.
func (c *Client) UpdateDataset(ctx context.Context, id string, req *DatasetUpdateRequest) (*DatasetResponse, error) {
	tflog.Trace(ctx, "UpdateDataset start")

	encodedID := url.PathEscape(id)
	resp, err := c.Put(ctx, "/pool/dataset/id/"+encodedID, req)
	if err != nil {
		return nil, fmt.Errorf("updating dataset %q: %w", id, err)
	}

	var dataset DatasetResponse
	if err := json.Unmarshal(resp, &dataset); err != nil {
		return nil, fmt.Errorf("parsing dataset update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateDataset success")
	return &dataset, nil
}

// DeleteDataset deletes a ZFS dataset.
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	tflog.Trace(ctx, "DeleteDataset start")

	encodedID := url.PathEscape(id)
	_, err := c.Delete(ctx, "/pool/dataset/id/"+encodedID)
	if err != nil {
		return fmt.Errorf("deleting dataset %q: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteDataset success")
	return nil
}
