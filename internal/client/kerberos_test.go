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

func TestKerberosRealm_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("List success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/realm") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.KerberosRealm{
				{ID: 1, Realm: "EXAMPLE.COM", KDC: []string{"kdc1.example.com"}},
				{ID: 2, Realm: "CORP.LOCAL"},
			})
		}))
		got, err := c.ListKerberosRealms(ctx)
		if err != nil {
			t.Fatalf("ListKerberosRealms: %v", err)
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
		_, err := c.ListKerberosRealms(ctx)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Get realm success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/kerberos/realm/id/1") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.KerberosRealm{
				ID:            1,
				Realm:         "EXAMPLE.COM",
				KDC:           []string{"kdc1", "kdc2"},
				AdminServer:   []string{"admin1"},
				KPasswdServer: []string{"kp1"},
			})
		}))
		got, err := c.GetKerberosRealm(ctx, 1)
		if err != nil {
			t.Fatalf("GetKerberosRealm: %v", err)
		}
		if got.Realm != "EXAMPLE.COM" {
			t.Errorf("Realm = %q", got.Realm)
		}
		if len(got.KDC) != 2 {
			t.Errorf("KDC len = %d", len(got.KDC))
		}
	})

	t.Run("Get realm 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetKerberosRealm(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get realm invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetKerberosRealm(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create realm success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.KerberosRealmCreateRequest
			_ = json.Unmarshal(body, &req)
			if req.Realm != "NEW.REALM" {
				t.Errorf("Realm = %q", req.Realm)
			}
			if len(req.KDC) != 1 || req.KDC[0] != "kdc.new.realm" {
				t.Errorf("KDC = %v", req.KDC)
			}
			writeJSON(w, http.StatusOK, client.KerberosRealm{ID: 10, Realm: "NEW.REALM"})
		}))
		got, err := c.CreateKerberosRealm(ctx, &client.KerberosRealmCreateRequest{
			Realm:         "NEW.REALM",
			KDC:           []string{"kdc.new.realm"},
			AdminServer:   []string{"admin.new.realm"},
			KPasswdServer: []string{"kp.new.realm"},
		})
		if err != nil {
			t.Fatalf("CreateKerberosRealm: %v", err)
		}
		if got.ID != 10 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create realm 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad realm"})
		}))
		_, err := c.CreateKerberosRealm(ctx, &client.KerberosRealmCreateRequest{Realm: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Update realm success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/realm/id/10") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["realm"] != "RENAMED.COM" {
				t.Errorf("realm = %v", raw["realm"])
			}
			if _, has := raw["kdc"]; has {
				t.Errorf("kdc should be omitted when nil")
			}
			writeJSON(w, http.StatusOK, client.KerberosRealm{ID: 10, Realm: "RENAMED.COM"})
		}))
		name := "RENAMED.COM"
		got, err := c.UpdateKerberosRealm(ctx, 10, &client.KerberosRealmUpdateRequest{Realm: &name})
		if err != nil {
			t.Fatalf("UpdateKerberosRealm: %v", err)
		}
		if got.Realm != "RENAMED.COM" {
			t.Errorf("Realm = %q", got.Realm)
		}
	})

	t.Run("Update realm 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		n := "x"
		_, err := c.UpdateKerberosRealm(ctx, 1, &client.KerberosRealmUpdateRequest{Realm: &n})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete realm success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/realm/id/10") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteKerberosRealm(ctx, 10); err != nil {
			t.Fatalf("DeleteKerberosRealm: %v", err)
		}
	})

	t.Run("Delete realm 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteKerberosRealm(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}

func TestKerberosKeytab_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get keytab success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/keytab/id/2") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.KerberosKeytab{
				ID:   2,
				Name: "ad-keytab",
				File: "BASE64DATA",
			})
		}))
		got, err := c.GetKerberosKeytab(ctx, 2)
		if err != nil {
			t.Fatalf("GetKerberosKeytab: %v", err)
		}
		if got.Name != "ad-keytab" || got.File != "BASE64DATA" {
			t.Errorf("got = %+v", got)
		}
	})

	t.Run("Get keytab 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetKerberosKeytab(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get keytab invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetKerberosKeytab(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create keytab success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/keytab") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.KerberosKeytabCreateRequest
			_ = json.Unmarshal(body, &req)
			if req.Name != "tab1" || req.File != "b64=" {
				t.Errorf("req = %+v", req)
			}
			writeJSON(w, http.StatusOK, client.KerberosKeytab{ID: 5, Name: "tab1"})
		}))
		got, err := c.CreateKerberosKeytab(ctx, &client.KerberosKeytabCreateRequest{
			Name: "tab1",
			File: "b64=",
		})
		if err != nil {
			t.Fatalf("CreateKerberosKeytab: %v", err)
		}
		if got.ID != 5 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create keytab 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid keytab"})
		}))
		_, err := c.CreateKerberosKeytab(ctx, &client.KerberosKeytabCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create keytab 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.CreateKerberosKeytab(ctx, &client.KerberosKeytabCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Update keytab success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/keytab/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["name"] != "newname" {
				t.Errorf("name = %v", raw["name"])
			}
			if _, has := raw["file"]; has {
				t.Errorf("file should be omitted when nil")
			}
			writeJSON(w, http.StatusOK, client.KerberosKeytab{ID: 5, Name: "newname"})
		}))
		n := "newname"
		got, err := c.UpdateKerberosKeytab(ctx, 5, &client.KerberosKeytabUpdateRequest{Name: &n})
		if err != nil {
			t.Fatalf("UpdateKerberosKeytab: %v", err)
		}
		if got.Name != "newname" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update keytab 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		n := "x"
		_, err := c.UpdateKerberosKeytab(ctx, 5, &client.KerberosKeytabUpdateRequest{Name: &n})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete keytab success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/kerberos/keytab/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteKerberosKeytab(ctx, 5); err != nil {
			t.Fatalf("DeleteKerberosKeytab: %v", err)
		}
	})

	t.Run("Delete keytab 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteKerberosKeytab(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
