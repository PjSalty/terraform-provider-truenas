package types

import (
	"encoding/json"
	"fmt"
)

// ISCSIExtent represents an iSCSI extent.
type ISCSIExtent struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Disk        json.RawMessage `json:"disk,omitempty"`
	Path        string          `json:"path,omitempty"`
	Filesize    json.RawMessage `json:"filesize,omitempty"`
	Blocksize   int             `json:"blocksize"`
	RPM         string          `json:"rpm,omitempty"`
	Enabled     bool            `json:"enabled"`
	Comment     string          `json:"comment,omitempty"`
	ReadOnly    bool            `json:"ro"`
	Xen         bool            `json:"xen"`
	InsecureTPC bool            `json:"insecure_tpc"`
}

// GetDisk returns the disk value as a string, handling null JSON values.
func (e *ISCSIExtent) GetDisk() string {
	if len(e.Disk) == 0 || string(e.Disk) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(e.Disk, &s); err != nil {
		return ""
	}
	return s
}

// GetFilesize returns the filesize as int64, handling both string and number JSON values.
func (e *ISCSIExtent) GetFilesize() int64 {
	if len(e.Filesize) == 0 || string(e.Filesize) == "null" {
		return 0
	}
	// Try as number first
	var n int64
	if err := json.Unmarshal(e.Filesize, &n); err == nil {
		return n
	}
	// Try as string
	var s string
	if err := json.Unmarshal(e.Filesize, &s); err == nil {
		var parsed int64
		if _, err := fmt.Sscanf(s, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

// ISCSIExtentCreateRequest represents the request to create an iSCSI extent.
type ISCSIExtentCreateRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Disk        string `json:"disk,omitempty"`
	Path        string `json:"path,omitempty"`
	Filesize    int64  `json:"filesize,omitempty"`
	Blocksize   int    `json:"blocksize"`
	RPM         string `json:"rpm,omitempty"`
	Enabled     bool   `json:"enabled"`
	Comment     string `json:"comment,omitempty"`
	ReadOnly    bool   `json:"ro"`
	Xen         bool   `json:"xen"`
	InsecureTPC bool   `json:"insecure_tpc"`
}

// ISCSIExtentUpdateRequest represents the request to update an iSCSI extent.
type ISCSIExtentUpdateRequest struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Disk      string `json:"disk,omitempty"`
	Path      string `json:"path,omitempty"`
	Filesize  int64  `json:"filesize,omitempty"`
	Blocksize int    `json:"blocksize,omitempty"`
	RPM       string `json:"rpm,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
	Comment   string `json:"comment,omitempty"`
	ReadOnly  *bool  `json:"ro,omitempty"`
}
