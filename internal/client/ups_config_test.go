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

func TestGetUPSConfig_Success(t *testing.T) {
	ctx := context.Background()
	want := client.UPSConfig{
		ID: 1, Mode: "MASTER", Identifier: "ups", Driver: "apcupsd",
		Port: "auto", RemoteHost: "", RemotePort: 3493, Shutdown: "BATT",
		ShutdownTimer: 30, Description: "main UPS",
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/ups") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetUPSConfig(ctx)
	if err != nil {
		t.Fatalf("GetUPSConfig: %v", err)
	}
	if got.Mode != "MASTER" {
		t.Errorf("Mode: %q", got.Mode)
	}
	if got.Driver != "apcupsd" {
		t.Errorf("Driver: %q", got.Driver)
	}
	if got.ShutdownTimer != 30 {
		t.Errorf("ShutdownTimer: %d", got.ShutdownTimer)
	}
}

func TestGetUPSConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetUPSConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetUPSConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.GetUPSConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing UPS config") {
		t.Errorf("got: %v", err)
	}
}

func TestGetUPSConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "boom"})
	}))
	_, err := c.GetUPSConfig(ctx)
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

func TestUpdateUPSConfig_Success(t *testing.T) {
	ctx := context.Background()
	mode := "SLAVE"
	identifier := "ups1"
	driver := "usbhid-ups"
	port := "usb"
	remoteHost := "10.0.0.1"
	remotePort := 3493
	shutdown := "LOWBATT"
	shutdownTimer := 60
	description := "new ups"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.UPSConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Mode == nil || *req.Mode != "SLAVE" {
			t.Errorf("Mode wrong")
		}
		if req.Driver == nil || *req.Driver != "usbhid-ups" {
			t.Errorf("Driver wrong")
		}
		if req.RemotePort == nil || *req.RemotePort != 3493 {
			t.Errorf("RemotePort wrong")
		}
		writeJSON(w, http.StatusOK, client.UPSConfig{
			ID: 1, Mode: "SLAVE", Identifier: "ups1", Driver: "usbhid-ups",
			Port: "usb", RemoteHost: "10.0.0.1", RemotePort: 3493,
			Shutdown: "LOWBATT", ShutdownTimer: 60, Description: "new ups",
		})
	}))

	got, err := c.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{
		Mode: &mode, Identifier: &identifier, Driver: &driver, Port: &port,
		RemoteHost: &remoteHost, RemotePort: &remotePort, Shutdown: &shutdown,
		ShutdownTimer: &shutdownTimer, Description: &description,
	})
	if err != nil {
		t.Fatalf("UpdateUPSConfig: %v", err)
	}
	if got.Mode != "SLAVE" {
		t.Errorf("Mode: %q", got.Mode)
	}
}

func TestUpdateUPSConfig_OmitEmpty(t *testing.T) {
	ctx := context.Background()
	mode := "MASTER"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "mode") {
			t.Errorf("missing mode: %s", body)
		}
		if strings.Contains(string(body), "driver") {
			t.Errorf("should omit driver: %s", body)
		}
		writeJSON(w, http.StatusOK, client.UPSConfig{ID: 1, Mode: "MASTER"})
	}))
	_, err := c.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{Mode: &mode})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestUpdateUPSConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{})
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

func TestUpdateUPSConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "junk")
	}))
	_, err := c.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing UPS config update") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateUPSConfig_404(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
	}))
	_, err := c.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetUPSConfig_Defaults(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.UPSConfig{ID: 5})
	}))
	got, err := c.GetUPSConfig(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != 5 || got.Mode != "" {
		t.Errorf("defaults: %+v", got)
	}
}

func TestGetUPSConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var method string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		writeJSON(w, http.StatusOK, client.UPSConfig{ID: 1})
	}))
	if _, err := c.GetUPSConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet {
		t.Errorf("method: %s", method)
	}
}

func TestUpdateUPSConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var method string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		writeJSON(w, http.StatusOK, client.UPSConfig{ID: 1})
	}))
	if _, err := c.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut {
		t.Errorf("method: %s", method)
	}
}
