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

func TestACMEDNSAuthenticator_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/acme/dns/authenticator/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.ACMEDNSAuthenticator{
				ID:   3,
				Name: "cloudflare-dns",
				Attributes: map[string]interface{}{
					"authenticator":        "cloudflare",
					"cloudflare_api_token": "token",
				},
			})
		}))

		got, err := c.GetACMEDNSAuthenticator(ctx, 3)
		if err != nil {
			t.Fatalf("GetACMEDNSAuthenticator: %v", err)
		}
		if got.ID != 3 || got.Name != "cloudflare-dns" {
			t.Errorf("got = %+v", got)
		}
		if got.Attributes["authenticator"] != "cloudflare" {
			t.Errorf("authenticator = %v", got.Attributes["authenticator"])
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetACMEDNSAuthenticator(ctx, 99)
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
		_, err := c.GetACMEDNSAuthenticator(ctx, 1)
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
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.GetACMEDNSAuthenticator(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/acme/dns/authenticator") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.ACMEDNSAuthenticatorCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "cf1" {
				t.Errorf("Name = %q", req.Name)
			}
			if req.Attributes["authenticator"] != "cloudflare" {
				t.Errorf("authenticator = %v", req.Attributes["authenticator"])
			}
			writeJSON(w, http.StatusOK, client.ACMEDNSAuthenticator{ID: 5, Name: "cf1"})
		}))

		got, err := c.CreateACMEDNSAuthenticator(ctx, &client.ACMEDNSAuthenticatorCreateRequest{
			Name: "cf1",
			Attributes: map[string]interface{}{
				"authenticator":        "cloudflare",
				"cloudflare_api_token": "tok",
			},
		})
		if err != nil {
			t.Fatalf("CreateACMEDNSAuthenticator: %v", err)
		}
		if got.ID != 5 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "missing attr"})
		}))
		_, err := c.CreateACMEDNSAuthenticator(ctx, &client.ACMEDNSAuthenticatorCreateRequest{Name: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Create 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.CreateACMEDNSAuthenticator(ctx, &client.ACMEDNSAuthenticatorCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.CreateACMEDNSAuthenticator(ctx, &client.ACMEDNSAuthenticatorCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/acme/dns/authenticator/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.ACMEDNSAuthenticatorUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "updated" {
				t.Errorf("Name = %q", req.Name)
			}
			writeJSON(w, http.StatusOK, client.ACMEDNSAuthenticator{ID: 5, Name: "updated"})
		}))
		got, err := c.UpdateACMEDNSAuthenticator(ctx, 5, &client.ACMEDNSAuthenticatorUpdateRequest{Name: "updated"})
		if err != nil {
			t.Fatalf("UpdateACMEDNSAuthenticator: %v", err)
		}
		if got.Name != "updated" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateACMEDNSAuthenticator(ctx, 5, &client.ACMEDNSAuthenticatorUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.UpdateACMEDNSAuthenticator(ctx, 5, &client.ACMEDNSAuthenticatorUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/acme/dns/authenticator/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteACMEDNSAuthenticator(ctx, 5); err != nil {
			t.Fatalf("DeleteACMEDNSAuthenticator: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteACMEDNSAuthenticator(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Delete 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		err := c.DeleteACMEDNSAuthenticator(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create attributes round-trip", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			attrs := raw["attributes"].(map[string]interface{})
			if attrs["route53_access_key_id"] != "AKIA" {
				t.Errorf("access_key_id = %v", attrs["route53_access_key_id"])
			}
			writeJSON(w, http.StatusOK, client.ACMEDNSAuthenticator{ID: 10, Name: "r53"})
		}))
		_, _ = c.CreateACMEDNSAuthenticator(ctx, &client.ACMEDNSAuthenticatorCreateRequest{
			Name: "r53",
			Attributes: map[string]interface{}{
				"authenticator":             "route53",
				"route53_access_key_id":     "AKIA",
				"route53_secret_access_key": "secret",
			},
		})
	})
}
