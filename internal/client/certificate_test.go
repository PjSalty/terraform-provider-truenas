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

func TestCertificate_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("List success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/certificate") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.Certificate{
				{ID: 1, Name: "cert1"},
				{ID: 2, Name: "cert2"},
			})
		}))
		got, err := c.ListCertificates(ctx)
		if err != nil {
			t.Fatalf("ListCertificates: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("len = %d", len(got))
		}
	})

	t.Run("List invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.ListCertificates(ctx)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/certificate/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Certificate{ID: 7, Name: "mycert", Common: "example.com"})
		}))
		got, err := c.GetCertificate(ctx, 7)
		if err != nil {
			t.Fatalf("GetCertificate: %v", err)
		}
		if got.ID != 7 || got.Common != "example.com" {
			t.Errorf("got = %+v", got)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))
		_, err := c.GetCertificate(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetCertificate(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("GetByName found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Certificate{
				{ID: 1, Name: "a"},
				{ID: 2, Name: "target"},
				{ID: 3, Name: "c"},
			})
		}))
		got, err := c.GetCertificateByName(ctx, "target")
		if err != nil {
			t.Fatalf("GetCertificateByName: %v", err)
		}
		if got.ID != 2 {
			t.Errorf("ID = %d, want 2", got.ID)
		}
	})

	t.Run("GetByName not found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Certificate{{ID: 1, Name: "a"}})
		}))
		_, err := c.GetCertificateByName(ctx, "missing")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false (err=%v)", err)
		}
	})

	t.Run("Create with job polling", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/certificate"):
				body, _ := io.ReadAll(r.Body)
				var req client.CertificateCreateRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if req.Name != "newcert" || req.CreateType != "CERTIFICATE_CREATE_IMPORTED" {
					t.Errorf("req = %+v", req)
				}
				_, _ = w.Write([]byte("42"))
			case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/core/get_jobs"):
				result, _ := json.Marshal(client.Certificate{ID: 10, Name: "newcert"})
				writeJSON(w, http.StatusOK, []client.Job{{
					ID:     42,
					State:  "SUCCESS",
					Result: result,
				}})
			default:
				t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			}
		}))

		got, err := c.CreateCertificate(ctx, &client.CertificateCreateRequest{
			Name:            "newcert",
			CreateType:      "CERTIFICATE_CREATE_IMPORTED",
			CertificateData: "-----BEGIN CERT-----",
			Privatekey:      "-----BEGIN KEY-----",
		})
		if err != nil {
			t.Fatalf("CreateCertificate: %v", err)
		}
		if got.ID != 10 || got.Name != "newcert" {
			t.Errorf("got = %+v", got)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad cert"})
		}))
		_, err := c.CreateCertificate(ctx, &client.CertificateCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Create job FAILED", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				_, _ = w.Write([]byte("7"))
				return
			}
			writeJSON(w, http.StatusOK, []client.Job{{ID: 7, State: "FAILED", Error: "bad"}})
		}))
		_, err := c.CreateCertificate(ctx, &client.CertificateCreateRequest{Name: "x", CreateType: "CERTIFICATE_CREATE_IMPORTED"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create invalid job ID response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"not":"an int"}`))
		}))
		_, err := c.CreateCertificate(ctx, &client.CertificateCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update direct cert response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/certificate/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.CertificateUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.Name != "renamed" {
				t.Errorf("Name = %q", req.Name)
			}
			writeJSON(w, http.StatusOK, client.Certificate{ID: 7, Name: "renamed"})
		}))
		got, err := c.UpdateCertificate(ctx, 7, &client.CertificateUpdateRequest{Name: "renamed"})
		if err != nil {
			t.Fatalf("UpdateCertificate: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update with job ID", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				_, _ = w.Write([]byte("5"))
				return
			}
			result, _ := json.Marshal(client.Certificate{ID: 7, Name: "via-job"})
			writeJSON(w, http.StatusOK, []client.Job{{ID: 5, State: "SUCCESS", Result: result}})
		}))
		got, err := c.UpdateCertificate(ctx, 7, &client.CertificateUpdateRequest{Name: "via-job"})
		if err != nil {
			t.Fatalf("UpdateCertificate: %v", err)
		}
		if got.Name != "via-job" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateCertificate(ctx, 7, &client.CertificateUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete success (non-job)", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteCertificate(ctx, 7); err != nil {
			t.Fatalf("DeleteCertificate: %v", err)
		}
	})

	t.Run("Delete with job ID", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte("9"))
				return
			}
			writeJSON(w, http.StatusOK, []client.Job{{ID: 9, State: "SUCCESS"}})
		}))
		if err := c.DeleteCertificate(ctx, 7); err != nil {
			t.Fatalf("DeleteCertificate: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteCertificate(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
