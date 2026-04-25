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

func TestGetFTPConfig_Success(t *testing.T) {
	ctx := context.Background()
	want := client.FTPConfig{
		ID: 1, Port: 21, Clients: 32, IPConnections: 5, LoginAttempt: 3,
		Timeout: 600, Banner: "Welcome", Filemask: "077", Dirmask: "077",
		FXP: true, Resume: true, DefaultRoot: true, TLS: false,
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/ftp") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))

	got, err := c.GetFTPConfig(ctx)
	if err != nil {
		t.Fatalf("GetFTPConfig: %v", err)
	}
	if got.Port != 21 {
		t.Errorf("Port: got %d", got.Port)
	}
	if !got.FXP || !got.Resume {
		t.Errorf("bools wrong")
	}
	if got.Banner != "Welcome" {
		t.Errorf("Banner: %q", got.Banner)
	}
}

func TestGetFTPConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "missing"})
	}))
	_, err := c.GetFTPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetFTPConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-json")
	}))
	_, err := c.GetFTPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing FTP config") {
		t.Errorf("got: %v", err)
	}
}

func TestGetFTPConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "boom"})
	}))
	_, err := c.GetFTPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("status: %d", apiErr.StatusCode)
	}
}

func TestUpdateFTPConfig_Success(t *testing.T) {
	ctx := context.Background()
	port := 2121
	clients := 100
	banner := "Test Banner"
	tls := true

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.FTPConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Port == nil || *req.Port != 2121 {
			t.Errorf("Port wrong")
		}
		if req.Clients == nil || *req.Clients != 100 {
			t.Errorf("Clients wrong")
		}
		if req.Banner == nil || *req.Banner != "Test Banner" {
			t.Errorf("Banner wrong")
		}
		if req.TLS == nil || !*req.TLS {
			t.Errorf("TLS wrong")
		}
		writeJSON(w, http.StatusOK, client.FTPConfig{ID: 1, Port: 2121, Clients: 100, Banner: "Test Banner", TLS: true})
	}))

	got, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{
		Port: &port, Clients: &clients, Banner: &banner, TLS: &tls,
	})
	if err != nil {
		t.Fatalf("UpdateFTPConfig: %v", err)
	}
	if got.Port != 2121 {
		t.Errorf("Port: %d", got.Port)
	}
}

func TestUpdateFTPConfig_OmitEmpty(t *testing.T) {
	ctx := context.Background()
	port := 21

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "port") {
			t.Errorf("missing port: %s", body)
		}
		if strings.Contains(string(body), "clients") {
			t.Errorf("should omit clients: %s", body)
		}
		writeJSON(w, http.StatusOK, client.FTPConfig{ID: 1, Port: 21})
	}))

	_, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{Port: &port})
	if err != nil {
		t.Fatalf("UpdateFTPConfig: %v", err)
	}
}

func TestUpdateFTPConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad port"})
	}))
	_, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "bad port" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestUpdateFTPConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "garbage")
	}))
	_, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing FTP config update") {
		t.Errorf("unexpected: %v", err)
	}
}

func TestUpdateFTPConfig_404(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetFTPConfig_EmptyDefaults(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.FTPConfig{ID: 1})
	}))
	got, err := c.GetFTPConfig(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Port != 0 || got.TLS {
		t.Errorf("zero expected")
	}
}

func TestUpdateFTPConfig_AllFieldsMarshaled(t *testing.T) {
	ctx := context.Background()
	p := 21
	cli := 10
	ipc := 5
	la := 3
	to := 600
	oa := true
	ol := true
	ban := "B"
	fm := "077"
	dm := "077"
	fxp := true
	res := true
	dr := true
	tls := true

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		for _, key := range []string{"port", "clients", "ipconnections", "loginattempt", "timeout",
			"onlyanonymous", "onlylocal", "banner", "filemask", "dirmask", "fxp", "resume", "defaultroot", "tls"} {
			if !strings.Contains(string(body), key) {
				t.Errorf("missing key %q", key)
			}
		}
		writeJSON(w, http.StatusOK, client.FTPConfig{ID: 1, Port: 21})
	}))

	_, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{
		Port: &p, Clients: &cli, IPConnections: &ipc, LoginAttempt: &la,
		Timeout: &to, OnlyAnonymous: &oa, OnlyLocal: &ol, Banner: &ban,
		Filemask: &fm, Dirmask: &dm, FXP: &fxp, Resume: &res,
		DefaultRoot: &dr, TLS: &tls,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestGetFTPConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var gotMethod string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		writeJSON(w, http.StatusOK, client.FTPConfig{ID: 1})
	}))
	if _, err := c.GetFTPConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method: %s", gotMethod)
	}
}

func TestUpdateFTPConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var gotMethod string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		writeJSON(w, http.StatusOK, client.FTPConfig{ID: 1})
	}))
	if _, err := c.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{}); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method: %s", gotMethod)
	}
}
