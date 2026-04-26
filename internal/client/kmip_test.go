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

func TestGetKMIPConfig_Success(t *testing.T) {
	ctx := context.Background()
	cert := 5
	ca := 10
	server := "kmip.example.com"
	want := client.KMIPConfig{
		ID: 1, Enabled: true, ManageSEDDisks: true, ManageZFSKeys: false,
		Certificate: &cert, CertificateAuthority: &ca, Port: 5696,
		Server: &server, SSLVersion: "PROTOCOL_TLSv1_2",
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/kmip") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetKMIPConfig(ctx)
	if err != nil {
		t.Fatalf("GetKMIPConfig: %v", err)
	}
	if !got.Enabled {
		t.Errorf("Enabled expected true")
	}
	if got.Port != 5696 {
		t.Errorf("Port: %d", got.Port)
	}
	if got.Server == nil || *got.Server != "kmip.example.com" {
		t.Errorf("Server wrong")
	}
	if got.Certificate == nil || *got.Certificate != 5 {
		t.Errorf("Certificate wrong")
	}
}

func TestGetKMIPConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetKMIPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetKMIPConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.GetKMIPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing KMIP config") {
		t.Errorf("got: %v", err)
	}
}

func TestGetKMIPConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "boom"})
	}))
	_, err := c.GetKMIPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status: %d", apiErr.StatusCode)
	}
}

func TestUpdateKMIPConfig_Success(t *testing.T) {
	ctx := context.Background()
	enabled := true
	manageSED := true
	port := 5696
	server := "kmip.new.com"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.KMIPUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Enabled == nil || !*req.Enabled {
			t.Errorf("Enabled wrong")
		}
		if req.ManageSEDDisks == nil || !*req.ManageSEDDisks {
			t.Errorf("ManageSEDDisks wrong")
		}
		if req.Port == nil || *req.Port != 5696 {
			t.Errorf("Port wrong")
		}
		writeJSON(w, http.StatusOK, client.KMIPConfig{
			ID: 1, Enabled: true, ManageSEDDisks: true, Port: 5696, Server: &server,
		})
	}))

	got, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{
		Enabled: &enabled, ManageSEDDisks: &manageSED, Port: &port, Server: &server,
	})
	if err != nil {
		t.Fatalf("UpdateKMIPConfig: %v", err)
	}
	if got.Port != 5696 {
		t.Errorf("Port: %d", got.Port)
	}
}

func TestUpdateKMIPConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "bad" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestUpdateKMIPConfig_RefetchOnZeroID(t *testing.T) {
	// If the PUT returns an object with ID=0 (job response), the client should
	// re-fetch the canonical state via GetKMIPConfig.
	ctx := context.Background()
	call := 0
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// First call = the PUT. Return a "job-ish" response (ID=0).
			if r.Method != http.MethodPut {
				t.Errorf("first call method: %s", r.Method)
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"id": 0})
			return
		}
		// Second call = re-fetch GET.
		if r.Method != http.MethodGet {
			t.Errorf("second call method: %s", r.Method)
		}
		writeJSON(w, http.StatusOK, client.KMIPConfig{ID: 42, Enabled: true, Port: 5696})
	}))

	got, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{})
	if err != nil {
		t.Fatalf("UpdateKMIPConfig: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID: got %d, want 42 (should have been refetched)", got.ID)
	}
	if call != 2 {
		t.Errorf("expected 2 calls, got %d", call)
	}
}

func TestUpdateKMIPConfig_RefetchFailure(t *testing.T) {
	ctx := context.Background()
	call := 0
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			writeJSON(w, http.StatusOK, map[string]interface{}{"id": 0})
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "refetch failed"})
	}))

	_, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateKMIPConfig_OmitEmpty(t *testing.T) {
	ctx := context.Background()
	enabled := true
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "enabled") {
			t.Errorf("missing enabled: %s", body)
		}
		if strings.Contains(string(body), "port") {
			t.Errorf("should omit port: %s", body)
		}
		writeJSON(w, http.StatusOK, client.KMIPConfig{ID: 1, Enabled: true})
	}))
	_, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{Enabled: &enabled})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestUpdateKMIPConfig_ForceClearField(t *testing.T) {
	ctx := context.Background()
	forceClear := true
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "force_clear") {
			t.Errorf("missing force_clear: %s", body)
		}
		writeJSON(w, http.StatusOK, client.KMIPConfig{ID: 1})
	}))
	_, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{ForceClear: &forceClear})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestUpdateKMIPConfig_404(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
	}))
	_, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetKMIPConfig_NilPointers(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"enabled":false,"certificate":null,"certificate_authority":null,"server":null,"port":5696,"ssl_version":"PROTOCOL_TLSv1_2","manage_sed_disks":false,"manage_zfs_keys":false}`))
	}))
	got, err := c.GetKMIPConfig(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Certificate != nil || got.CertificateAuthority != nil || got.Server != nil {
		t.Errorf("expected nil pointers: %+v", got)
	}
}

func TestUpdateKMIPConfig_AllFields(t *testing.T) {
	ctx := context.Background()
	enabled := true
	sed := true
	zfs := true
	cert := 1
	ca := 2
	port := 5696
	server := "s"
	sslv := "PROTOCOL_TLSv1_2"
	fc := true
	cs := true
	val := true

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		for _, key := range []string{"enabled", "manage_sed_disks", "manage_zfs_keys",
			"certificate", "certificate_authority", "port", "server", "ssl_version",
			"force_clear", "change_server", "validate"} {
			if !strings.Contains(string(body), key) {
				t.Errorf("missing key %q", key)
			}
		}
		writeJSON(w, http.StatusOK, client.KMIPConfig{ID: 1})
	}))
	_, err := c.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{
		Enabled: &enabled, ManageSEDDisks: &sed, ManageZFSKeys: &zfs,
		Certificate: &cert, CertificateAuthority: &ca, Port: &port,
		Server: &server, SSLVersion: &sslv,
		ForceClear: &fc, ChangeServer: &cs, Validate: &val,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
