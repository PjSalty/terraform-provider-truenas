package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// appRouter serves /app, /app/id/{id}, and /core/get_jobs to support
// Create/Update/Delete flows that use WaitForJob internally.
type appRouter struct {
	t          *testing.T
	onGet      func(r *http.Request) (interface{}, int)
	onCreate   func(r *http.Request) (interface{}, int)
	onUpdate   func(r *http.Request) (interface{}, int)
	onDelete   func(r *http.Request) (interface{}, int)
	onList     func(r *http.Request) (interface{}, int)
	jobState   string // "SUCCESS", "FAILED"
	jobResult  json.RawMessage
	jobError   string
	jobCallCnt int32
}

func (h *appRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.Contains(path, "/core/get_jobs") && r.Method == http.MethodGet:
		atomic.AddInt32(&h.jobCallCnt, 1)
		state := h.jobState
		if state == "" {
			state = "SUCCESS"
		}
		writeJSON(w, http.StatusOK, []client.Job{{
			ID:     1,
			State:  state,
			Result: h.jobResult,
			Error:  h.jobError,
		}})
	case strings.Contains(path, "/app/id/") && r.Method == http.MethodGet:
		if h.onGet == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "no onGet"})
			return
		}
		obj, status := h.onGet(r)
		writeJSON(w, status, obj)
	case strings.Contains(path, "/app/id/") && r.Method == http.MethodPut:
		if h.onUpdate == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "no onUpdate"})
			return
		}
		obj, status := h.onUpdate(r)
		writeJSON(w, status, obj)
	case strings.Contains(path, "/app/id/") && r.Method == http.MethodDelete:
		if h.onDelete == nil {
			writeJSON(w, http.StatusOK, 1)
			return
		}
		obj, status := h.onDelete(r)
		writeJSON(w, status, obj)
	case strings.HasSuffix(path, "/app") && r.Method == http.MethodPost:
		if h.onCreate == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "no onCreate"})
			return
		}
		obj, status := h.onCreate(r)
		writeJSON(w, status, obj)
	case strings.HasSuffix(path, "/app") && r.Method == http.MethodGet:
		if h.onList == nil {
			writeJSON(w, http.StatusOK, []client.App{})
			return
		}
		obj, status := h.onList(r)
		writeJSON(w, status, obj)
	default:
		h.t.Errorf("unexpected request: %s %s", r.Method, path)
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "router: no route"})
	}
}

func TestApp_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetApp success (bare object)", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onGet: func(r *http.Request) (interface{}, int) {
				if !strings.HasSuffix(r.URL.Path, "/app/id/jellyfin") {
					t.Errorf("path = %s", r.URL.Path)
				}
				return client.App{
					ID: "jellyfin", Name: "jellyfin", State: "RUNNING",
					Version: "10.9.0", HumanVersion: "10.9.0_2.3.1",
				}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		got, err := c.GetApp(ctx, "jellyfin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "jellyfin" || got.State != "RUNNING" {
			t.Errorf("unexpected: %+v", got)
		}
	})

	t.Run("GetApp success (list response)", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onGet: func(_ *http.Request) (interface{}, int) {
				return []client.App{{ID: "jellyfin", Name: "jellyfin", State: "RUNNING"}}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		got, err := c.GetApp(ctx, "jellyfin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "jellyfin" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("GetApp empty list => 404", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onGet: func(_ *http.Request) (interface{}, int) {
				return []client.App{}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		_, err := c.GetApp(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("GetApp 404", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onGet: func(_ *http.Request) (interface{}, int) {
				return map[string]string{"message": "not found"}, http.StatusNotFound
			},
		}
		_, c := newTestServer(t, router)

		_, err := c.GetApp(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("GetApp invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.GetApp(ctx, "x")
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetApp URL encodes id", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onGet: func(r *http.Request) (interface{}, int) {
				// url.PathEscape turns no special chars; use special ones to verify.
				if r.URL.RawPath != "" && !strings.Contains(r.URL.RawPath, "my%2Fapp") {
					t.Errorf("raw path = %s", r.URL.RawPath)
				}
				return client.App{ID: "my/app", Name: "my/app"}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		_, err := c.GetApp(ctx, "my/app")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ListApps success", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onList: func(_ *http.Request) (interface{}, int) {
				return []client.App{
					{ID: "a1", Name: "a1", State: "RUNNING"},
					{ID: "a2", Name: "a2", State: "STOPPED"},
				}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		list, err := c.ListApps(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("ListApps invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/app") {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("bogus"))
			}
		}))
		_, err := c.ListApps(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("CreateApp job success", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onCreate: func(r *http.Request) (interface{}, int) {
				body, _ := io.ReadAll(r.Body)
				var decoded map[string]interface{}
				if err := json.Unmarshal(body, &decoded); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if decoded["app_name"].(string) != "jellyfin" {
					t.Errorf("app_name = %v", decoded["app_name"])
				}
				if decoded["catalog_app"].(string) != "jellyfin" {
					t.Errorf("catalog_app = %v", decoded["catalog_app"])
				}
				return 1, http.StatusOK // job ID
			},
			onGet: func(_ *http.Request) (interface{}, int) {
				return client.App{ID: "jellyfin", Name: "jellyfin", State: "RUNNING"}, http.StatusOK
			},
			jobState: "SUCCESS",
		}
		_, c := newTestServer(t, router)

		got, err := c.CreateApp(ctx, &client.AppCreateRequest{
			AppName:    "jellyfin",
			CatalogApp: "jellyfin",
			Train:      "stable",
			Version:    "10.9.0",
			Values:     map[string]interface{}{"replicas": 1},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "jellyfin" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("CreateApp job failed", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onCreate: func(_ *http.Request) (interface{}, int) {
				return 1, http.StatusOK
			},
			jobState: "FAILED",
			jobError: "install timeout",
		}
		_, c := newTestServer(t, router)

		_, err := c.CreateApp(ctx, &client.AppCreateRequest{AppName: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "install timeout") && !strings.Contains(err.Error(), "failed") {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("CreateApp 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/app") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))

		_, err := c.CreateApp(ctx, &client.AppCreateRequest{AppName: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("CreateApp invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.CreateApp(ctx, &client.AppCreateRequest{AppName: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateApp job success", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onUpdate: func(r *http.Request) (interface{}, int) {
				if !strings.HasSuffix(r.URL.Path, "/app/id/jellyfin") {
					t.Errorf("path = %s", r.URL.Path)
				}
				return 2, http.StatusOK
			},
			onGet: func(_ *http.Request) (interface{}, int) {
				return client.App{ID: "jellyfin", Name: "jellyfin", State: "RUNNING"}, http.StatusOK
			},
			jobState: "SUCCESS",
		}
		_, c := newTestServer(t, router)

		got, err := c.UpdateApp(ctx, "jellyfin", &client.AppUpdateRequest{
			Values: map[string]interface{}{"replicas": 2},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "jellyfin" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("UpdateApp 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad values"})
		}))
		withFastRetries(c, 1)

		_, err := c.UpdateApp(ctx, "jellyfin", &client.AppUpdateRequest{
			Values: map[string]interface{}{"x": 1},
		})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("DeleteApp job success", func(t *testing.T) {
		router := &appRouter{
			t: t,
			onDelete: func(r *http.Request) (interface{}, int) {
				body, _ := io.ReadAll(r.Body)
				var decoded map[string]interface{}
				if err := json.Unmarshal(body, &decoded); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if decoded["remove_images"].(bool) != true {
					t.Errorf("remove_images = %v", decoded["remove_images"])
				}
				return 3, http.StatusOK
			},
			jobState: "SUCCESS",
		}
		_, c := newTestServer(t, router)

		if err := c.DeleteApp(ctx, "jellyfin", &client.AppDeleteRequest{
			RemoveImages:    true,
			RemoveIxVolumes: false,
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteApp 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		withFastRetries(c, 1)

		err := c.DeleteApp(ctx, "jellyfin", &client.AppDeleteRequest{})
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}

func TestCatalog_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetCatalog success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/catalog") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Catalog{
				ID:              "TRUENAS",
				Label:           "TRUENAS",
				PreferredTrains: []string{"stable", "community"},
				Location:        "/mnt/.ix-apps/truenas_catalog",
			})
		}))

		got, err := c.GetCatalog(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Label != "TRUENAS" {
			t.Errorf("Label = %q", got.Label)
		}
		if len(got.PreferredTrains) != 2 {
			t.Errorf("PreferredTrains = %+v", got.PreferredTrains)
		}
	})

	t.Run("GetCatalog invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bogus"))
		}))
		_, err := c.GetCatalog(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateCatalog sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "preferred_trains") {
				t.Errorf("body missing preferred_trains: %s", body)
			}
			writeJSON(w, http.StatusOK, client.Catalog{
				ID: "TRUENAS", PreferredTrains: []string{"stable"},
			})
		}))

		trains := []string{"stable"}
		got, err := c.UpdateCatalog(ctx, &client.CatalogUpdateRequest{
			PreferredTrains: &trains,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.PreferredTrains) != 1 {
			t.Errorf("PreferredTrains = %+v", got.PreferredTrains)
		}
	})
}
