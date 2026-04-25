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

func TestGetStaticRoute_Success(t *testing.T) {
	ctx := context.Background()
	want := client.StaticRoute{
		ID: 1, Destination: "10.0.0.0/24", Gateway: "192.168.1.1", Description: "LAN",
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/staticroute/id/1") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetStaticRoute(ctx, 1)
	if err != nil {
		t.Fatalf("GetStaticRoute: %v", err)
	}
	if got.Destination != "10.0.0.0/24" {
		t.Errorf("Destination: %q", got.Destination)
	}
	if got.Gateway != "192.168.1.1" {
		t.Errorf("Gateway: %q", got.Gateway)
	}
}

func TestGetStaticRoute_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetStaticRoute(ctx, 99)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetStaticRoute_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xx")
	}))
	_, err := c.GetStaticRoute(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing static route") {
		t.Errorf("got: %v", err)
	}
}

func TestGetStaticRoute_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.GetStaticRoute(ctx, 1)
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

func TestCreateStaticRoute_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/staticroute") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.StaticRouteCreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Destination != "10.1.0.0/16" {
			t.Errorf("Destination: %q", req.Destination)
		}
		if req.Gateway != "10.0.0.1" {
			t.Errorf("Gateway: %q", req.Gateway)
		}
		if req.Description != "test" {
			t.Errorf("Description: %q", req.Description)
		}
		writeJSON(w, http.StatusOK, client.StaticRoute{
			ID: 5, Destination: "10.1.0.0/16", Gateway: "10.0.0.1", Description: "test",
		})
	}))

	got, err := c.CreateStaticRoute(ctx, &client.StaticRouteCreateRequest{
		Destination: "10.1.0.0/16", Gateway: "10.0.0.1", Description: "test",
	})
	if err != nil {
		t.Fatalf("CreateStaticRoute: %v", err)
	}
	if got.ID != 5 {
		t.Errorf("ID: %d", got.ID)
	}
}

func TestCreateStaticRoute_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid gw"})
	}))
	_, err := c.CreateStaticRoute(ctx, &client.StaticRouteCreateRequest{
		Destination: "x", Gateway: "y",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "invalid gw" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestCreateStaticRoute_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "garbage")
	}))
	_, err := c.CreateStaticRoute(ctx, &client.StaticRouteCreateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing static route create") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateStaticRoute_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/staticroute/id/3") {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.StaticRouteUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Gateway != "10.0.0.2" {
			t.Errorf("Gateway: %q", req.Gateway)
		}
		writeJSON(w, http.StatusOK, client.StaticRoute{
			ID: 3, Destination: "10.1.0.0/16", Gateway: "10.0.0.2",
		})
	}))

	got, err := c.UpdateStaticRoute(ctx, 3, &client.StaticRouteUpdateRequest{
		Gateway: "10.0.0.2",
	})
	if err != nil {
		t.Fatalf("UpdateStaticRoute: %v", err)
	}
	if got.Gateway != "10.0.0.2" {
		t.Errorf("Gateway: %q", got.Gateway)
	}
}

func TestUpdateStaticRoute_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateStaticRoute(ctx, 1, &client.StaticRouteUpdateRequest{})
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

func TestUpdateStaticRoute_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "bogus")
	}))
	_, err := c.UpdateStaticRoute(ctx, 1, &client.StaticRouteUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing static route update") {
		t.Errorf("got: %v", err)
	}
}

func TestDeleteStaticRoute_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/staticroute/id/7") {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.DeleteStaticRoute(ctx, 7); err != nil {
		t.Fatalf("DeleteStaticRoute: %v", err)
	}
}

func TestDeleteStaticRoute_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
	}))
	err := c.DeleteStaticRoute(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestDeleteStaticRoute_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "locked"})
	}))
	err := c.DeleteStaticRoute(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "locked" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestCreateStaticRoute_OmitEmptyDescription(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "description") {
			t.Errorf("description should be omitted: %s", body)
		}
		writeJSON(w, http.StatusOK, client.StaticRoute{ID: 1, Destination: "x", Gateway: "y"})
	}))
	_, err := c.CreateStaticRoute(ctx, &client.StaticRouteCreateRequest{
		Destination: "x", Gateway: "y",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
