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

func TestListServices_Success(t *testing.T) {
	ctx := context.Background()
	want := []client.Service{
		{ID: 1, Service: "ssh", Enable: true, State: "RUNNING", Pids: []int{100, 101}},
		{ID: 2, Service: "ftp", Enable: false, State: "STOPPED", Pids: nil},
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len: %d", len(got))
	}
	if got[0].Service != "ssh" {
		t.Errorf("Service[0]: %q", got[0].Service)
	}
	if !got[0].Enable || got[1].Enable {
		t.Errorf("enable fields wrong")
	}
}

func TestListServices_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xx")
	}))
	_, err := c.ListServices(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing services list") {
		t.Errorf("got: %v", err)
	}
}

func TestListServices_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.ListServices(ctx)
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

func TestGetService_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/id/5") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.Service{
			ID: 5, Service: "nfs", Enable: true, State: "RUNNING",
		})
	}))
	got, err := c.GetService(ctx, 5)
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if got.Service != "nfs" {
		t.Errorf("Service: %q", got.Service)
	}
}

func TestGetService_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetService(ctx, 99)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetService_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xx")
	}))
	_, err := c.GetService(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing service") {
		t.Errorf("got: %v", err)
	}
}

func TestGetServiceByName_Found(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Service{
			{ID: 1, Service: "ssh", Enable: true},
			{ID: 2, Service: "ftp", Enable: false},
		})
	}))
	got, err := c.GetServiceByName(ctx, "ftp")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("ID: %d", got.ID)
	}
}

func TestGetServiceByName_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Service{
			{ID: 1, Service: "ssh"},
		})
	}))
	_, err := c.GetServiceByName(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	// It should return an *APIError with 404 status.
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("status: %d", apiErr.StatusCode)
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound")
	}
}

func TestGetServiceByName_ListError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.GetServiceByName(ctx, "ssh")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateService_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/id/3") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.ServiceUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !req.Enable {
			t.Errorf("Enable should be true")
		}
		writeJSON(w, http.StatusOK, client.Service{ID: 3, Enable: true})
	}))
	if err := c.UpdateService(ctx, 3, &client.ServiceUpdateRequest{Enable: true}); err != nil {
		t.Fatalf("UpdateService: %v", err)
	}
}

func TestUpdateService_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	err := c.UpdateService(ctx, 1, &client.ServiceUpdateRequest{Enable: false})
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

func TestStartService_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/start") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.ServiceStartStopRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Service != "ssh" {
			t.Errorf("Service: %q", req.Service)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`true`))
	}))
	if err := c.StartService(ctx, "ssh"); err != nil {
		t.Fatalf("StartService: %v", err)
	}
}

func TestStartService_Error(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "fail"})
	}))
	err := c.StartService(ctx, "ssh")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "starting service") {
		t.Errorf("got: %v", err)
	}
}

func TestStopService_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/stop") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.ServiceStartStopRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Service != "ftp" {
			t.Errorf("Service: %q", req.Service)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`true`))
	}))
	if err := c.StopService(ctx, "ftp"); err != nil {
		t.Fatalf("StopService: %v", err)
	}
}

func TestStopService_Error(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "fail"})
	}))
	err := c.StopService(ctx, "ftp")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "stopping service") {
		t.Errorf("got: %v", err)
	}
}

func TestGetService_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.GetService(ctx, 1)
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

func TestListServices_Empty(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Service{})
	}))
	got, err := c.ListServices(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty: %+v", got)
	}
}
