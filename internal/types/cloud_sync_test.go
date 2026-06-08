package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCloudSync_UnmarshalJSON_NestedCredentialsObject mirrors the
// internal/client test from PR #12 against the shared types.CloudSync.
// TrueNAS returns credentials as a nested object on GET / list and as a
// plain integer on create / update — the unmarshaler must accept both.
func TestCloudSync_UnmarshalJSON_NestedCredentialsObject(t *testing.T) {
	body := `{
        "id": 2,
        "description": "Google Drive - Max",
        "path": "/mnt/Data/backup/max_google-drive",
        "credentials": {
            "id": 2,
            "name": "Google Drive",
            "provider": {"type": "GOOGLE_DRIVE"}
        },
        "direction": "PULL",
        "transfer_mode": "SYNC",
        "schedule": {"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*"},
        "enabled": true,
        "attributes": {"folder": "/", "fast_list": false, "acknowledge_abuse": false}
    }`

	var cs CloudSync
	if err := json.Unmarshal([]byte(body), &cs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Credentials != 2 {
		t.Errorf("Credentials = %d, want 2", cs.Credentials)
	}
	if cs.Description != "Google Drive - Max" {
		t.Errorf("Description = %q", cs.Description)
	}
	if cs.Direction != "PULL" {
		t.Errorf("Direction = %q", cs.Direction)
	}
	if !cs.Enabled {
		t.Error("Enabled = false, want true")
	}
}

// TestCloudSync_UnmarshalJSON_PlainIntCredentials covers the
// create / update response shape, where credentials is the plain integer.
func TestCloudSync_UnmarshalJSON_PlainIntCredentials(t *testing.T) {
	body := `{"id": 3, "path": "/mnt/x", "credentials": 7, "direction": "PUSH", "transfer_mode": "COPY", "enabled": true, "schedule": {"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*"}}`

	var cs CloudSync
	if err := json.Unmarshal([]byte(body), &cs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Credentials != 7 {
		t.Errorf("Credentials = %d, want 7", cs.Credentials)
	}
}

// TestCloudSync_UnmarshalJSON_MissingCredentials accepts payloads
// where credentials is absent — leaves the field as the zero value.
func TestCloudSync_UnmarshalJSON_MissingCredentials(t *testing.T) {
	body := `{"id": 4, "path": "/mnt/q"}`

	var cs CloudSync
	if err := json.Unmarshal([]byte(body), &cs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Credentials != 0 {
		t.Errorf("Credentials = %d, want 0", cs.Credentials)
	}
	if cs.Path != "/mnt/q" {
		t.Errorf("Path = %q", cs.Path)
	}
}

// TestCloudSync_UnmarshalJSON_MalformedCredentials returns an error
// rather than silently accepting garbage (e.g. credentials: "string").
func TestCloudSync_UnmarshalJSON_MalformedCredentials(t *testing.T) {
	body := `{"id": 5, "credentials": "not-a-number-not-an-object"}`

	var cs CloudSync
	err := json.Unmarshal([]byte(body), &cs)
	if err == nil {
		t.Fatal("expected error on string credentials, got nil")
	}
	if !strings.Contains(err.Error(), "credentials field") {
		t.Errorf("expected wrapped 'credentials field' error, got: %v", err)
	}
}

// TestCloudSync_UnmarshalJSON_DirectCallMalformed covers the inner
// json.Unmarshal(data, aux) error path. Reachable only by calling
// UnmarshalJSON directly with garbage — the std-lib parser would
// reject malformed bytes before delegating to a custom UnmarshalJSON.
// Callers that re-use the method on a stream of bytes (e.g. testing
// harnesses) still get a clean error rather than a silent partial
// state.
func TestCloudSync_UnmarshalJSON_DirectCallMalformed(t *testing.T) {
	var cs CloudSync
	err := cs.UnmarshalJSON([]byte(`{this is not valid`))
	if err == nil {
		t.Fatal("expected error on direct call with malformed bytes")
	}
}
