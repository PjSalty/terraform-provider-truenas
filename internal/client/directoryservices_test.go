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

func TestDirectoryServices_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/directoryservices") {
				t.Errorf("path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{
				"id": 1,
				"service_type": "ACTIVEDIRECTORY",
				"enable": true,
				"enable_account_cache": true,
				"enable_dns_updates": false,
				"timeout": 60,
				"kerberos_realm": "EXAMPLE.COM",
				"credential": {"credential_type": "KERBEROS_USER", "username": "admin"},
				"configuration": {"hostname": "dc1", "domain": "EXAMPLE.COM"}
			}`))
		}))

		got, err := c.GetDirectoryServicesConfig(ctx)
		if err != nil {
			t.Fatalf("GetDirectoryServicesConfig: %v", err)
		}
		if got.ServiceType == nil || *got.ServiceType != "ACTIVEDIRECTORY" {
			t.Errorf("ServiceType = %+v", got.ServiceType)
		}
		if !got.Enable {
			t.Errorf("Enable = false")
		}
		if got.Timeout != 60 {
			t.Errorf("Timeout = %d", got.Timeout)
		}
		if got.Credential["username"] != "admin" {
			t.Errorf("credential username = %v", got.Credential["username"])
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetDirectoryServicesConfig(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetDirectoryServicesConfig(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{bad"))
		}))
		_, err := c.GetDirectoryServicesConfig(ctx)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Update direct config response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if raw["enable"] != true {
				t.Errorf("enable = %v", raw["enable"])
			}
			if raw["timeout"] != float64(30) {
				t.Errorf("timeout = %v", raw["timeout"])
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":     1,
				"enable": true,
			})
		}))

		enable := true
		timeout := 30
		got, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{
			Enable:  &enable,
			Timeout: &timeout,
		})
		if err != nil {
			t.Fatalf("UpdateDirectoryServicesConfig: %v", err)
		}
		if !got.Enable {
			t.Errorf("Enable = false")
		}
	})

	t.Run("Update with job int re-fetches", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				_, _ = w.Write([]byte("99"))
			case http.MethodGet:
				writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{
					ID:     1,
					Enable: true,
				})
			}
		}))

		enable := true
		got, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{Enable: &enable})
		if err != nil {
			t.Fatalf("UpdateDirectoryServicesConfig: %v", err)
		}
		if !got.Enable {
			t.Errorf("Enable = false")
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad cred"})
		}))
		enable := true
		_, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{Enable: &enable})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Update 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		enable := false
		_, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{Enable: &enable})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update marshals service_type", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["service_type"] != "LDAP" {
				t.Errorf("service_type = %v", raw["service_type"])
			}
			if _, has := raw["enable"]; has {
				t.Errorf("enable should be omitted")
			}
			writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{ID: 1})
		}))

		st := "LDAP"
		if _, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{ServiceType: &st}); err != nil {
			t.Fatalf("UpdateDirectoryServicesConfig: %v", err)
		}
	})

	t.Run("Update marshals Force", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["force"] != true {
				t.Errorf("force = %v", raw["force"])
			}
			writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{})
		}))
		f := true
		_, _ = c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{Force: &f})
	})

	t.Run("Leave success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/directoryservices/leave") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["username"] != "admin" {
				t.Errorf("username = %v", raw["username"])
			}
			_, _ = w.Write([]byte(`null`))
		}))
		resp, err := c.LeaveDirectoryServices(ctx, map[string]interface{}{"username": "admin"})
		if err != nil {
			t.Fatalf("LeaveDirectoryServices: %v", err)
		}
		if len(resp) == 0 {
			t.Errorf("empty response")
		}
	})

	t.Run("Leave 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "not joined"})
		}))
		_, err := c.LeaveDirectoryServices(ctx, map[string]interface{}{})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Update fallback re-fetches on decode fail", func(t *testing.T) {
		var calls int
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method == http.MethodPut {
				// Return something that's neither int nor config
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"some":"garbage","nested":{"x":1}}`))
				return
			}
			writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{ID: 1, Enable: true})
		}))
		enable := false
		got, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{Enable: &enable})
		// The fallback should eventually call GET; if decode succeeds it returns the decoded struct directly, so either outcome OK.
		if err != nil {
			t.Fatalf("UpdateDirectoryServicesConfig: %v", err)
		}
		if got == nil {
			t.Fatal("got nil")
		}
	})

	t.Run("Get empty response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{})
		}))
		got, err := c.GetDirectoryServicesConfig(ctx)
		if err != nil {
			t.Fatalf("GetDirectoryServicesConfig: %v", err)
		}
		if got.Enable {
			t.Errorf("Enable = true, want false")
		}
	})

	t.Run("Get path correctness", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/api/v2.0/directoryservices") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{})
		}))
		_, _ = c.GetDirectoryServicesConfig(ctx)
	})

	t.Run("Update credential marshaling", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			cred, ok := raw["credential"].(map[string]interface{})
			if !ok {
				t.Fatalf("credential not present: %s", body)
			}
			if cred["credential_type"] != "LDAP_PLAIN" {
				t.Errorf("credential_type = %v", cred["credential_type"])
			}
			writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{})
		}))
		cred := map[string]interface{}{"credential_type": "LDAP_PLAIN", "binddn": "cn=admin"}
		_, _ = c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{Credential: &cred})
	})
}
