package wsclient

import (
	"strings"
	"testing"
)

// A replication task carries the key of its encrypted datasets. It must not
// reach a diagnostic or a log through an echoed request or response body.
func TestRedact_ReplicationEncryptionKey(t *testing.T) {
	if !isSensitiveKey("encryption_key") {
		t.Fatal("encryption_key is not treated as sensitive")
	}
	got := string(redactJSONBody([]byte(`{"name":"rep","encryption_key":"correct horse battery staple"}`)))
	if strings.Contains(got, "correct horse") {
		t.Errorf("the key survived redaction: %s", got)
	}
	if !strings.Contains(got, `"name":"rep"`) {
		t.Errorf("an unrelated field was lost: %s", got)
	}
}
