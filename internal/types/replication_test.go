package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// ReplicationEntry returns ssh_credentials as the expanded
// KeychainCredentialEntry (middleware api/v25_10_0/replication.py), not the
// ID that create and update take.
func TestReplication_UnmarshalJSON_SSHCredentialsObject(t *testing.T) {
	body := `{
		"id": 7, "name": "rep", "direction": "PUSH", "transport": "SSH",
		"source_datasets": ["test/src"], "target_dataset": "backup/src",
		"ssh_credentials": {
			"id": 175, "name": "acctest-sshc", "type": "SSH_CREDENTIALS",
			"attributes": {"host": "192.0.2.10", "port": 22, "private_key": 174}
		}
	}`
	var r Replication
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.SSHCredentials != 175 {
		t.Errorf("SSHCredentials = %d, want 175", r.SSHCredentials)
	}
	if r.ID != 7 || r.Name != "rep" || r.Transport != "SSH" || r.TargetDataset != "backup/src" {
		t.Errorf("other fields lost: %+v", r)
	}
}

func TestReplication_UnmarshalJSON_SSHCredentialsShapes(t *testing.T) {
	cases := map[string]struct {
		field string
		want  int
	}{
		"plain id": {`"ssh_credentials": 5,`, 5},
		"null":     {`"ssh_credentials": null,`, 0},
		"absent":   {``, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var r Replication
			if err := json.Unmarshal([]byte(`{`+tc.field+`"id": 1}`), &r); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.SSHCredentials != tc.want || r.ID != 1 {
				t.Errorf("got SSHCredentials=%d ID=%d, want %d and 1", r.SSHCredentials, r.ID, tc.want)
			}
		})
	}
}

func TestReplication_UnmarshalJSON_Errors(t *testing.T) {
	var r Replication
	if err := json.Unmarshal([]byte(`{"ssh_credentials": "x"}`), &r); err == nil ||
		!strings.Contains(err.Error(), "ssh_credentials") {
		t.Errorf("a string ssh_credentials gave %v, want an error naming the field", err)
	}
	if err := json.Unmarshal([]byte(`{"id": "not a number"}`), &r); err == nil {
		t.Error("a malformed body was accepted")
	}
}
