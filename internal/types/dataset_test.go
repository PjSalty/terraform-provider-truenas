package types

import (
	"encoding/json"
	"testing"
)

func TestDatasetResponse_GetVolsize_unset(t *testing.T) {
	d := &DatasetResponse{}
	if got := d.GetVolsize(); got != 0 {
		t.Errorf("unset: got %d, want 0", got)
	}
}

func TestDatasetResponse_GetVolsize_null(t *testing.T) {
	d := &DatasetResponse{Volsize: &PropertyRawVal{Rawvalue: ""}}
	if got := d.GetVolsize(); got != 0 {
		t.Errorf("empty rawvalue: got %d, want 0", got)
	}
}

func TestDatasetResponse_GetVolsize_valid(t *testing.T) {
	d := &DatasetResponse{Volsize: &PropertyRawVal{Rawvalue: "16777216"}}
	if got := d.GetVolsize(); got != 16777216 {
		t.Errorf("got %d, want 16777216", got)
	}
}

func TestDatasetResponse_GetVolsize_invalid(t *testing.T) {
	d := &DatasetResponse{Volsize: &PropertyRawVal{Rawvalue: "not-a-number"}}
	if got := d.GetVolsize(); got != 0 {
		t.Errorf("invalid: got %d, want 0", got)
	}
}

func TestDatasetResponse_GetVolblocksize_unset(t *testing.T) {
	d := &DatasetResponse{}
	if got := d.GetVolblocksize(); got != "" {
		t.Errorf("unset: got %q, want empty", got)
	}
}

func TestDatasetResponse_GetVolblocksize_set(t *testing.T) {
	d := &DatasetResponse{Volblocksize: &PropertyValue{Value: "16K"}}
	if got := d.GetVolblocksize(); got != "16K" {
		t.Errorf("got %q, want 16K", got)
	}
}

func TestDatasetResponse_GetComments_userPropertiesPath(t *testing.T) {
	d := &DatasetResponse{
		UserProperties: map[string]*PropertyValue{
			"comments": {Value: "from-user-properties"},
		},
	}
	if got := d.GetComments(); got != "from-user-properties" {
		t.Errorf("got %q, want from-user-properties", got)
	}
}

func TestDatasetResponse_GetComments_userPropertiesEmpty_FallsBack(t *testing.T) {
	d := &DatasetResponse{
		UserProperties: map[string]*PropertyValue{
			"comments": {Value: ""},
		},
		Comments: &PropertyValue{Value: "fallback-top-level"},
	}
	if got := d.GetComments(); got != "fallback-top-level" {
		t.Errorf("got %q, want fallback-top-level", got)
	}
}

func TestDatasetResponse_GetComments_userPropertiesNil_FallsBack(t *testing.T) {
	d := &DatasetResponse{
		UserProperties: map[string]*PropertyValue{"comments": nil},
		Comments:       &PropertyValue{Value: "top-level"},
	}
	if got := d.GetComments(); got != "top-level" {
		t.Errorf("got %q, want top-level", got)
	}
}

func TestDatasetResponse_GetComments_topLevelOnly(t *testing.T) {
	d := &DatasetResponse{
		Comments: &PropertyValue{Value: "only-top-level"},
	}
	if got := d.GetComments(); got != "only-top-level" {
		t.Errorf("got %q, want only-top-level", got)
	}
}

func TestDatasetResponse_GetComments_empty(t *testing.T) {
	d := &DatasetResponse{}
	if got := d.GetComments(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestDatasetResponse_GetComments_topLevelEmpty(t *testing.T) {
	d := &DatasetResponse{Comments: &PropertyValue{Value: ""}}
	if got := d.GetComments(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// Smoke-test that DatasetResponse round-trips a typical TrueNAS payload
// without panicking, exercises the full struct shape including pointer
// PropertyValue fields and nested user_properties.
func TestDatasetResponse_unmarshalSmoke(t *testing.T) {
	body := []byte(`{
		"id": "tank/data",
		"name": "tank/data",
		"pool": "tank",
		"type": "FILESYSTEM",
		"mountpoint": "/mnt/tank/data",
		"comments": null,
		"user_properties": {
			"comments": {"value": "owned by ops", "source": "LOCAL", "parsed": "owned by ops"}
		},
		"compression": {"value": "LZ4", "source": "LOCAL", "parsed": "lz4"},
		"volsize": null,
		"volblocksize": null
	}`)
	var d DatasetResponse
	if err := json.Unmarshal(body, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.GetComments() != "owned by ops" {
		t.Errorf("GetComments: got %q", d.GetComments())
	}
	if d.GetVolsize() != 0 {
		t.Errorf("GetVolsize zvol-absent: got %d", d.GetVolsize())
	}
	if d.GetVolblocksize() != "" {
		t.Errorf("GetVolblocksize zvol-absent: got %q", d.GetVolblocksize())
	}
}
