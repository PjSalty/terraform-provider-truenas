package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestISCSIPortal_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.ISCSIPortal{
				ID:      1,
				Tag:     1,
				Comment: "primary",
				Listen: []client.ISCSIPortalListen{
					{IP: "10.10.40.10", Port: 3260},
				},
			})
		}))

		got, err := c.GetISCSIPortal(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Comment != "primary" {
			t.Errorf("Comment = %q", got.Comment)
		}
		if len(got.Listen) != 1 || got.Listen[0].IP != "10.10.40.10" {
			t.Errorf("Listen = %+v", got.Listen)
		}
		// Port is preserved in read responses (legacy compat).
		if got.Listen[0].Port != 3260 {
			t.Errorf("Port = %d, want 3260", got.Listen[0].Port)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))

		_, err := c.GetISCSIPortal(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("Create strips port from wire payload", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			raw := string(body)

			// CRITICAL: SCALE 25.10 rejects the "port" key with HTTP 422.
			// The MarshalJSON must NOT include it.
			if strings.Contains(raw, `"port"`) {
				t.Errorf("request body contains forbidden 'port' key: %s", raw)
			}
			// Sanity: the IP IS present.
			if !strings.Contains(raw, `"ip"`) {
				t.Errorf("request body missing 'ip' key: %s", raw)
			}
			// Decode to confirm structural shape.
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			listen, ok := decoded["listen"].([]interface{})
			if !ok {
				t.Fatalf("listen is not an array: %T", decoded["listen"])
			}
			if len(listen) != 2 {
				t.Errorf("listen entries = %d, want 2", len(listen))
			}
			for i, entry := range listen {
				m, ok := entry.(map[string]interface{})
				if !ok {
					t.Errorf("entry %d not object", i)
					continue
				}
				if _, has := m["port"]; has {
					t.Errorf("entry %d contains port: %+v", i, m)
				}
				if _, has := m["ip"]; !has {
					t.Errorf("entry %d missing ip: %+v", i, m)
				}
			}

			writeJSON(w, http.StatusOK, client.ISCSIPortal{
				ID:  5,
				Tag: 1,
				Listen: []client.ISCSIPortalListen{
					{IP: "10.10.40.10", Port: 3260},
					{IP: "10.10.40.11", Port: 3260},
				},
			})
		}))

		resp, err := c.CreateISCSIPortal(ctx, &client.ISCSIPortalCreateRequest{
			Listen: []client.ISCSIPortalListen{
				{IP: "10.10.40.10", Port: 3260},
				{IP: "10.10.40.11", Port: 3260},
			},
			Comment: "dual",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 5 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update strips port from wire payload", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), `"port"`) {
				t.Errorf("update body contains 'port': %s", body)
			}
			writeJSON(w, http.StatusOK, client.ISCSIPortal{ID: 5, Tag: 1})
		}))

		_, err := c.UpdateISCSIPortal(ctx, 5, &client.ISCSIPortalUpdateRequest{
			Listen: []client.ISCSIPortalListen{
				{IP: "10.10.40.10", Port: 3260},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteISCSIPortal(ctx, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestISCSIPortalCreateRequest_MarshalJSON directly exercises MarshalJSON
// without a server, for fast assertion on wire format.
func TestISCSIPortalCreateRequest_MarshalJSON(t *testing.T) {
	req := client.ISCSIPortalCreateRequest{
		Listen: []client.ISCSIPortalListen{
			{IP: "10.0.0.1", Port: 3260},
			{IP: "10.0.0.2", Port: 9999},
		},
		Comment: "test",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "port") {
		t.Errorf("JSON contains 'port': %s", s)
	}
	if strings.Contains(s, "3260") || strings.Contains(s, "9999") {
		t.Errorf("JSON contains port numbers: %s", s)
	}
	if !strings.Contains(s, `"ip":"10.0.0.1"`) {
		t.Errorf("JSON missing ip 10.0.0.1: %s", s)
	}
	if !strings.Contains(s, `"comment":"test"`) {
		t.Errorf("JSON missing comment: %s", s)
	}
}

func TestISCSIPortalUpdateRequest_MarshalJSON(t *testing.T) {
	req := client.ISCSIPortalUpdateRequest{
		Listen: []client.ISCSIPortalListen{
			{IP: "10.0.0.5", Port: 3260},
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "port") {
		t.Errorf("JSON contains 'port': %s", data)
	}
}
