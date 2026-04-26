package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestInitScript_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetInitScript success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/initshutdownscript/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.InitScript{
				ID: 7, Type: "COMMAND", Command: "/usr/bin/backup",
				When: "POSTINIT", Enabled: true, Timeout: 30,
			})
		}))
		got, err := c.GetInitScript(ctx, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Type != "COMMAND" {
			t.Errorf("Type = %q", got.Type)
		}
		if got.When != "POSTINIT" {
			t.Errorf("When = %q", got.When)
		}
	})

	t.Run("GetInitScript 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetInitScript(ctx, 999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetInitScript invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<bad>"))
		}))
		_, err := c.GetInitScript(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetInitScript server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetInitScript(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("CreateInitScript sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/initshutdownscript") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.InitScriptCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Type != "COMMAND" {
				t.Errorf("Type = %q", req.Type)
			}
			if req.Command != "/bin/hello" {
				t.Errorf("Command = %q", req.Command)
			}
			if req.When != "POSTINIT" {
				t.Errorf("When = %q", req.When)
			}
			writeJSON(w, http.StatusOK, client.InitScript{
				ID: 11, Type: req.Type, Command: req.Command,
				When: req.When, Enabled: req.Enabled,
			})
		}))
		resp, err := c.CreateInitScript(ctx, &client.InitScriptCreateRequest{
			Type: "COMMAND", Command: "/bin/hello",
			When: "POSTINIT", Enabled: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 11 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("CreateInitScript 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad type"})
		}))
		_, err := c.CreateInitScript(ctx, &client.InitScriptCreateRequest{Type: "BAD"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("CreateInitScript invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.CreateInitScript(ctx, &client.InitScriptCreateRequest{Type: "COMMAND"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateInitScript sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/initshutdownscript/id/11") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.InitScriptUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Command != "/bin/new" {
				t.Errorf("Command = %q", req.Command)
			}
			if req.Enabled == nil || *req.Enabled != false {
				t.Errorf("Enabled = %v, want false", req.Enabled)
			}
			writeJSON(w, http.StatusOK, client.InitScript{ID: 11, Command: "/bin/new"})
		}))
		enabled := false
		resp, err := c.UpdateInitScript(ctx, 11, &client.InitScriptUpdateRequest{
			Command: "/bin/new", Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Command != "/bin/new" {
			t.Errorf("Command = %q", resp.Command)
		}
	})

	t.Run("UpdateInitScript server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateInitScript(ctx, 1, &client.InitScriptUpdateRequest{Command: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("UpdateInitScript invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.UpdateInitScript(ctx, 1, &client.InitScriptUpdateRequest{Command: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("DeleteInitScript success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/initshutdownscript/id/11") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteInitScript(ctx, 11); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteInitScript 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteInitScript(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("DeleteInitScript server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "err"})
		}))
		err := c.DeleteInitScript(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateInitScript SCRIPT type", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.InitScriptCreateRequest
			_ = json.Unmarshal(body, &req)
			if req.Script != "/mnt/tank/script.sh" {
				t.Errorf("Script = %q", req.Script)
			}
			writeJSON(w, http.StatusOK, client.InitScript{ID: 20, Type: "SCRIPT", Script: req.Script})
		}))
		resp, err := c.CreateInitScript(ctx, &client.InitScriptCreateRequest{
			Type: "SCRIPT", Script: "/mnt/tank/script.sh", When: "PREINIT", Enabled: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Type != "SCRIPT" {
			t.Errorf("Type = %q", resp.Type)
		}
	})
}
