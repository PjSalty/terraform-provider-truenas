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

func TestAlertService_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get 25.10 polymorphic schema", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// 25.10+: type lives inside attributes as the discriminator.
			_, _ = w.Write([]byte(`{
				"id": 1,
				"name": "slack-alerts",
				"enabled": true,
				"level": "WARNING",
				"attributes": {
					"type": "SlackAlert",
					"cluster_name": "homelab",
					"url": "https://hooks.slack.com/services/xxx"
				}
			}`))
		}))

		got, err := c.GetAlertService(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "slack-alerts" {
			t.Errorf("Name = %q", got.Name)
		}
		if got.GetType() != "SlackAlert" {
			t.Errorf("GetType() = %q, want SlackAlert", got.GetType())
		}
		if got.Settings["url"] != "https://hooks.slack.com/services/xxx" {
			t.Errorf("url = %v", got.Settings["url"])
		}
	})

	t.Run("Get 25.04 legacy top-level type", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 2,
				"name": "legacy",
				"type": "Mail",
				"enabled": true,
				"level": "INFO",
				"attributes": {"email": "root@example.com"}
			}`))
		}))

		got, err := c.GetAlertService(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.GetType() != "Mail" {
			t.Errorf("GetType() = %q, want Mail", got.GetType())
		}
	})

	t.Run("GetType attributes preferred over top-level", func(t *testing.T) {
		svc := client.AlertService{
			Type:     "LegacyType",
			Settings: map[string]interface{}{"type": "NewType"},
		}
		if svc.GetType() != "NewType" {
			t.Errorf("GetType() = %q, want NewType", svc.GetType())
		}
	})

	t.Run("GetType empty", func(t *testing.T) {
		svc := client.AlertService{}
		if svc.GetType() != "" {
			t.Errorf("GetType() = %q, want empty", svc.GetType())
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))
		_, err := c.GetAlertService(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})

	t.Run("Create - no top-level type in body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			// 25.10+: top-level "type" field must NOT be in the create body
			// (AlertServiceCreateRequest has no Type field at all).
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if _, has := raw["type"]; has {
				t.Errorf("create body has forbidden top-level 'type': %s", body)
			}
			attrs, ok := raw["attributes"].(map[string]interface{})
			if !ok {
				t.Fatalf("attributes missing or wrong type: %T", raw["attributes"])
			}
			if attrs["type"] != "SlackAlert" {
				t.Errorf("attributes.type = %v, want SlackAlert", attrs["type"])
			}
			_, _ = w.Write([]byte(`{
				"id": 5,
				"name": "slack-new",
				"enabled": true,
				"level": "WARNING",
				"attributes": {"type": "SlackAlert", "url": "x"}
			}`))
		}))

		resp, err := c.CreateAlertService(ctx, &client.AlertServiceCreateRequest{
			Name:    "slack-new",
			Enabled: true,
			Level:   "WARNING",
			Settings: map[string]interface{}{
				"type": "SlackAlert",
				"url":  "x",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 5 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Create 422 rejects extra field", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"message": "Extra inputs are not permitted",
			})
		}))

		_, err := c.CreateAlertService(ctx, &client.AlertServiceCreateRequest{
			Name: "bad",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "Extra inputs") {
			t.Errorf("error = %v, want contains 'Extra inputs'", err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{
				"id": 5,
				"name": "slack-updated",
				"enabled": false,
				"level": "CRITICAL",
				"attributes": {"type": "SlackAlert"}
			}`))
		}))

		enabled := false
		resp, err := c.UpdateAlertService(ctx, 5, &client.AlertServiceUpdateRequest{
			Name:    "slack-updated",
			Enabled: &enabled,
			Level:   "CRITICAL",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "slack-updated" || resp.Enabled {
			t.Errorf("resp = %+v", resp)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteAlertService(ctx, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
