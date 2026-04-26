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

// ---------------------------------------------------------------------------
// GetUpdateAutoDownload
// ---------------------------------------------------------------------------

func TestGetUpdateAutoDownload_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/update/get_auto_download") {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "true")
	}))
	got, err := c.GetUpdateAutoDownload(ctx)
	if err != nil {
		t.Fatalf("GetUpdateAutoDownload: %v", err)
	}
	if !got {
		t.Errorf("got false, want true")
	}
}

func TestGetUpdateAutoDownload_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetUpdateAutoDownload(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetUpdateAutoDownload_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-a-bool")
	}))
	_, err := c.GetUpdateAutoDownload(ctx)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing update auto_download") {
		t.Errorf("got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SetUpdateAutoDownload
// ---------------------------------------------------------------------------

func TestSetUpdateAutoDownload_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/update/set_auto_download") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != "true" {
			t.Errorf("body: %q", string(body))
		}
		writeJSON(w, http.StatusOK, nil)
	}))
	if err := c.SetUpdateAutoDownload(ctx, true); err != nil {
		t.Fatalf("SetUpdateAutoDownload: %v", err)
	}
}

func TestSetUpdateAutoDownload_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))
	err := c.SetUpdateAutoDownload(ctx, false)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
}

// ---------------------------------------------------------------------------
// GetUpdateTrains
// ---------------------------------------------------------------------------

func TestGetUpdateTrains_Success(t *testing.T) {
	ctx := context.Background()
	want := client.UpdateTrains{
		Trains: map[string]client.UpdateTrainInfo{
			"TrueNAS-SCALE-Fangtooth": {Description: "Fangtooth 25.04 [release]"},
			"TrueNAS-SCALE-Goldeye":   {Description: "Goldeye 25.10"},
		},
		Current:  "TrueNAS-SCALE-Fangtooth",
		Selected: "TrueNAS-SCALE-Fangtooth",
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/update/get_trains") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetUpdateTrains(ctx)
	if err != nil {
		t.Fatalf("GetUpdateTrains: %v", err)
	}
	if got.Selected != "TrueNAS-SCALE-Fangtooth" {
		t.Errorf("Selected: %q", got.Selected)
	}
	if got.Current != "TrueNAS-SCALE-Fangtooth" {
		t.Errorf("Current: %q", got.Current)
	}
	if len(got.Trains) != 2 {
		t.Errorf("Trains len: %d", len(got.Trains))
	}
	if got.Trains["TrueNAS-SCALE-Goldeye"].Description != "Goldeye 25.10" {
		t.Errorf("Goldeye description wrong")
	}
}

func TestGetUpdateTrains_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.GetUpdateTrains(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetUpdateTrains_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.GetUpdateTrains(ctx)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing update trains") {
		t.Errorf("got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SetUpdateTrain
// ---------------------------------------------------------------------------

func TestSetUpdateTrain_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/update/set_train") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `"TrueNAS-SCALE-Fangtooth"` {
			t.Errorf("body: %q", string(body))
		}
		writeJSON(w, http.StatusOK, nil)
	}))
	if err := c.SetUpdateTrain(ctx, "TrueNAS-SCALE-Fangtooth"); err != nil {
		t.Fatalf("SetUpdateTrain: %v", err)
	}
}

func TestSetUpdateTrain_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "fail"})
	}))
	err := c.SetUpdateTrain(ctx, "whatever")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// CheckUpdateAvailable
// ---------------------------------------------------------------------------

func TestCheckUpdateAvailable_Unavailable(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/update/check_available") {
			t.Errorf("path: %s", r.URL.Path)
		}
		// Verify the request body is the expected empty-train object.
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req["train"] != "" {
			t.Errorf("train field should be omitted or empty, got: %q", req["train"])
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "UNAVAILABLE"})
	}))
	got, err := c.CheckUpdateAvailable(ctx)
	if err != nil {
		t.Fatalf("CheckUpdateAvailable: %v", err)
	}
	if got.Status != "UNAVAILABLE" {
		t.Errorf("Status: %q", got.Status)
	}
	if got.Version != "" {
		t.Errorf("Version should be empty, got: %q", got.Version)
	}
}

func TestCheckUpdateAvailable_Available(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "AVAILABLE",
			"version": "25.04.3.0",
			"changes": []map[string]string{{"operation_type": "upgrade", "old": "25.04.2.6", "new": "25.04.3.0"}},
			"notes":   "security patch",
		})
	}))
	got, err := c.CheckUpdateAvailable(ctx)
	if err != nil {
		t.Fatalf("CheckUpdateAvailable: %v", err)
	}
	if got.Status != "AVAILABLE" {
		t.Errorf("Status: %q", got.Status)
	}
	if got.Version != "25.04.3.0" {
		t.Errorf("Version: %q", got.Version)
	}
	if got.Notes != "security patch" {
		t.Errorf("Notes: %q", got.Notes)
	}
	if len(got.Changes) == 0 {
		t.Error("Changes should be populated")
	}
}

func TestCheckUpdateAvailable_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "oops"})
	}))
	_, err := c.CheckUpdateAvailable(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckUpdateAvailable_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-json")
	}))
	_, err := c.CheckUpdateAvailable(ctx)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing update check_available") {
		t.Errorf("got: %v", err)
	}
}
