package types

import (
	"encoding/json"
	"testing"
)

func TestISCSIExtent_GetDisk_unset(t *testing.T) {
	e := &ISCSIExtent{}
	if got := e.GetDisk(); got != "" {
		t.Errorf("unset: got %q", got)
	}
}

func TestISCSIExtent_GetDisk_null(t *testing.T) {
	e := &ISCSIExtent{Disk: json.RawMessage(`null`)}
	if got := e.GetDisk(); got != "" {
		t.Errorf("null: got %q", got)
	}
}

func TestISCSIExtent_GetDisk_string(t *testing.T) {
	e := &ISCSIExtent{Disk: json.RawMessage(`"zvol/tank/vm1"`)}
	if got := e.GetDisk(); got != "zvol/tank/vm1" {
		t.Errorf("string: got %q", got)
	}
}

func TestISCSIExtent_GetDisk_invalidJSON(t *testing.T) {
	e := &ISCSIExtent{Disk: json.RawMessage(`{not-valid}`)}
	if got := e.GetDisk(); got != "" {
		t.Errorf("invalid json: got %q", got)
	}
}

func TestISCSIExtent_GetFilesize_unset(t *testing.T) {
	e := &ISCSIExtent{}
	if got := e.GetFilesize(); got != 0 {
		t.Errorf("unset: got %d", got)
	}
}

func TestISCSIExtent_GetFilesize_null(t *testing.T) {
	e := &ISCSIExtent{Filesize: json.RawMessage(`null`)}
	if got := e.GetFilesize(); got != 0 {
		t.Errorf("null: got %d", got)
	}
}

func TestISCSIExtent_GetFilesize_number(t *testing.T) {
	e := &ISCSIExtent{Filesize: json.RawMessage(`16777216`)}
	if got := e.GetFilesize(); got != 16777216 {
		t.Errorf("number: got %d", got)
	}
}

func TestISCSIExtent_GetFilesize_stringNumeric(t *testing.T) {
	e := &ISCSIExtent{Filesize: json.RawMessage(`"16777216"`)}
	if got := e.GetFilesize(); got != 16777216 {
		t.Errorf("string-numeric: got %d", got)
	}
}

func TestISCSIExtent_GetFilesize_stringNonNumeric(t *testing.T) {
	e := &ISCSIExtent{Filesize: json.RawMessage(`"not-a-number"`)}
	if got := e.GetFilesize(); got != 0 {
		t.Errorf("string-non-numeric: got %d", got)
	}
}

func TestISCSIExtent_GetFilesize_object(t *testing.T) {
	// Neither int nor string parses; should return 0.
	e := &ISCSIExtent{Filesize: json.RawMessage(`{"foo":"bar"}`)}
	if got := e.GetFilesize(); got != 0 {
		t.Errorf("object: got %d", got)
	}
}
