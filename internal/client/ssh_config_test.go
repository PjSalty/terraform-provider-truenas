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

func TestGetSSHConfig_Success(t *testing.T) {
	ctx := context.Background()
	want := client.SSHConfig{
		ID:              1,
		TCPPort:         22,
		PasswordAuth:    true,
		KerberosAuth:    false,
		TCPFwd:          true,
		Compression:     false,
		SFTPLogLevel:    "INFO",
		SFTPLogFacility: "AUTH",
		WeakCiphers:     []string{"AES128-CBC"},
	}

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/ssh") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))

	got, err := c.GetSSHConfig(ctx)
	if err != nil {
		t.Fatalf("GetSSHConfig: %v", err)
	}
	if got.TCPPort != 22 {
		t.Errorf("TCPPort: got %d, want 22", got.TCPPort)
	}
	if !got.PasswordAuth {
		t.Error("PasswordAuth: expected true")
	}
	if got.SFTPLogLevel != "INFO" {
		t.Errorf("SFTPLogLevel: got %q", got.SFTPLogLevel)
	}
	if len(got.WeakCiphers) != 1 || got.WeakCiphers[0] != "AES128-CBC" {
		t.Errorf("WeakCiphers: got %v", got.WeakCiphers)
	}
}

func TestGetSSHConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
	}))

	_, err := c.GetSSHConfig(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestGetSSHConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "{not valid json")
	}))

	_, err := c.GetSSHConfig(ctx)
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing SSH config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSSHConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "boom"})
	}))

	_, err := c.GetSSHConfig(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode: got %d", apiErr.StatusCode)
	}
}

func TestUpdateSSHConfig_Success(t *testing.T) {
	ctx := context.Background()
	port := 2222
	passAuth := false
	compression := true
	weak := []string{"AES128-CBC"}

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: got %s, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.SSHConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.TCPPort == nil || *req.TCPPort != 2222 {
			t.Errorf("TCPPort in body wrong")
		}
		if req.PasswordAuth == nil || *req.PasswordAuth {
			t.Errorf("PasswordAuth in body wrong")
		}
		if req.Compression == nil || !*req.Compression {
			t.Errorf("Compression in body wrong")
		}
		writeJSON(w, http.StatusOK, client.SSHConfig{
			ID:           1,
			TCPPort:      2222,
			PasswordAuth: false,
			Compression:  true,
			WeakCiphers:  []string{"AES128-CBC"},
		})
	}))

	got, err := c.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{
		TCPPort:      &port,
		PasswordAuth: &passAuth,
		Compression:  &compression,
		WeakCiphers:  &weak,
	})
	if err != nil {
		t.Fatalf("UpdateSSHConfig: %v", err)
	}
	if got.TCPPort != 2222 {
		t.Errorf("TCPPort: got %d", got.TCPPort)
	}
}

func TestUpdateSSHConfig_OmitEmptyFields(t *testing.T) {
	ctx := context.Background()
	port := 22

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Only the TCPPort field should appear in the JSON body.
		if !strings.Contains(string(body), "tcpport") {
			t.Errorf("body missing tcpport: %s", body)
		}
		if strings.Contains(string(body), "passwordauth") {
			t.Errorf("body should omit passwordauth: %s", body)
		}
		writeJSON(w, http.StatusOK, client.SSHConfig{ID: 1, TCPPort: 22})
	}))

	_, err := c.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{TCPPort: &port})
	if err != nil {
		t.Fatalf("UpdateSSHConfig: %v", err)
	}
}

func TestUpdateSSHConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad port"})
	}))

	_, err := c.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode: got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "bad port" {
		t.Errorf("Message: got %q", apiErr.Message)
	}
}

func TestUpdateSSHConfig_InvalidJSONResponse(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "nope")
	}))

	_, err := c.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing SSH config update") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSSHConfig_EmptyResponse(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.SSHConfig{ID: 1})
	}))

	got, err := c.GetSSHConfig(ctx)
	if err != nil {
		t.Fatalf("GetSSHConfig: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("ID: got %d", got.ID)
	}
	if got.TCPPort != 0 {
		t.Errorf("TCPPort zero expected, got %d", got.TCPPort)
	}
}

func TestUpdateSSHConfig_404ReturnsError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))

	_, err := c.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetSSHConfig_AllFieldsDecoded(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.SSHConfig{
			ID:              9,
			TCPPort:         2022,
			PasswordAuth:    true,
			KerberosAuth:    true,
			TCPFwd:          true,
			Compression:     true,
			SFTPLogLevel:    "DEBUG",
			SFTPLogFacility: "LOCAL0",
			WeakCiphers:     []string{"a", "b"},
		})
	}))
	got, err := c.GetSSHConfig(ctx)
	if err != nil {
		t.Fatalf("GetSSHConfig: %v", err)
	}
	if got.ID != 9 || got.KerberosAuth != true || got.SFTPLogFacility != "LOCAL0" || len(got.WeakCiphers) != 2 {
		t.Errorf("fields decoded incorrectly: %+v", got)
	}
}

func TestUpdateSSHConfig_RoundTripAllFields(t *testing.T) {
	ctx := context.Background()
	tcp := 2022
	pw := true
	kb := true
	fwd := true
	cmp := true
	lvl := "DEBUG"
	fac := "LOCAL0"
	ciphers := []string{"x"}

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		for _, key := range []string{"tcpport", "passwordauth", "kerberosauth", "tcpfwd", "compression", "sftp_log_level", "sftp_log_facility", "weak_ciphers"} {
			if !strings.Contains(string(body), key) {
				t.Errorf("missing key %q in body: %s", key, body)
			}
		}
		writeJSON(w, http.StatusOK, client.SSHConfig{ID: 1, TCPPort: 2022})
	}))

	_, err := c.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{
		TCPPort: &tcp, PasswordAuth: &pw, KerberosAuth: &kb, TCPFwd: &fwd,
		Compression: &cmp, SFTPLogLevel: &lvl, SFTPLogFacility: &fac, WeakCiphers: &ciphers,
	})
	if err != nil {
		t.Fatalf("UpdateSSHConfig: %v", err)
	}
}
