package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestISCSIPortalCreateRequest_MarshalJSON_dropsPort(t *testing.T) {
	r := ISCSIPortalCreateRequest{
		Listen: []ISCSIPortalListen{
			{IP: "0.0.0.0", Port: 3260},
			{IP: "::", Port: 4420},
		},
		Comment: "primary",
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"ip":"0.0.0.0"`) {
		t.Errorf("missing ip 0.0.0.0: %s", s)
	}
	if strings.Contains(s, `"port"`) {
		t.Errorf("port leaked into write payload: %s", s)
	}
	if !strings.Contains(s, `"comment":"primary"`) {
		t.Errorf("missing comment: %s", s)
	}
}

func TestISCSIPortalCreateRequest_MarshalJSON_emptyListen(t *testing.T) {
	r := ISCSIPortalCreateRequest{}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"listen":[]`) {
		t.Errorf("expected empty listen array, got %s", string(b))
	}
}

func TestISCSIPortalUpdateRequest_MarshalJSON_dropsPort(t *testing.T) {
	r := ISCSIPortalUpdateRequest{
		Listen:  []ISCSIPortalListen{{IP: "10.0.0.1", Port: 3260}},
		Comment: "updated",
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if strings.Contains(s, `"port"`) {
		t.Errorf("port leaked into update payload: %s", s)
	}
	if !strings.Contains(s, `"ip":"10.0.0.1"`) {
		t.Errorf("missing ip: %s", s)
	}
}

func TestISCSIPortalUpdateRequest_MarshalJSON_emptyListenOmitted(t *testing.T) {
	// Empty listen on update should be omitted (omitempty),
	// allowing a comment-only update.
	r := ISCSIPortalUpdateRequest{Comment: "comment-only-change"}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if strings.Contains(s, `"listen"`) {
		t.Errorf("expected listen to be omitted: %s", s)
	}
	if !strings.Contains(s, `"comment":"comment-only-change"`) {
		t.Errorf("missing comment: %s", s)
	}
}
