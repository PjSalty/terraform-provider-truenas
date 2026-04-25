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

func TestSystemDataset_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/systemdataset") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.SystemDataset{
				ID:       1,
				Pool:     "tank",
				PoolSet:  true,
				UUID:     "uuid-1",
				Basename: "tank/.system",
				Path:     "/var/db/system",
			})
		}))

		got, err := c.GetSystemDataset(ctx)
		if err != nil {
			t.Fatalf("GetSystemDataset: %v", err)
		}
		if got.Pool != "tank" {
			t.Errorf("Pool = %q", got.Pool)
		}
		if !got.PoolSet {
			t.Errorf("PoolSet = false")
		}
		if got.UUID != "uuid-1" {
			t.Errorf("UUID = %q", got.UUID)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))
		_, err := c.GetSystemDataset(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetSystemDataset(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err = %T", err)
		}
		if apiErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.GetSystemDataset(ctx)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Update direct config response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.SystemDatasetUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decode: %v", err)
			}
			if req.Pool == nil || *req.Pool != "tank" {
				t.Errorf("Pool = %+v", req.Pool)
			}
			// Return a config object directly (not a job ID)
			writeJSON(w, http.StatusOK, client.SystemDataset{
				ID:   1,
				Pool: "tank",
			})
		}))

		pool := "tank"
		got, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{Pool: &pool})
		if err != nil {
			t.Fatalf("UpdateSystemDataset: %v", err)
		}
		if got.Pool != "tank" {
			t.Errorf("Pool = %q", got.Pool)
		}
	})

	t.Run("Update with job ID polls and re-fetches", func(t *testing.T) {
		var phase atomic.Int32
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/systemdataset"):
				// Return a bare int job ID.
				_, _ = w.Write([]byte("42"))
			case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{
					ID:    42,
					State: "SUCCESS",
				}})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/systemdataset"):
				writeJSON(w, http.StatusOK, client.SystemDataset{ID: 1, Pool: "newpool"})
			default:
				t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			}
			phase.Add(1)
		}))

		newpool := "newpool"
		got, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{Pool: &newpool})
		if err != nil {
			t.Fatalf("UpdateSystemDataset: %v", err)
		}
		if got.Pool != "newpool" {
			t.Errorf("Pool = %q", got.Pool)
		}
	})

	t.Run("Update 422 error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid pool"})
		}))
		pool := "bad"
		_, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{Pool: &pool})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err = %T", err)
		}
		if apiErr.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("Update 500 error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		pool := "tank"
		_, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{Pool: &pool})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update body marshals PoolExclude", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if raw["pool_exclude"] != "boot-pool" {
				t.Errorf("pool_exclude = %v", raw["pool_exclude"])
			}
			if _, has := raw["pool"]; has {
				t.Errorf("pool should be omitted when nil")
			}
			writeJSON(w, http.StatusOK, client.SystemDataset{ID: 1})
		}))
		ex := "boot-pool"
		if _, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{PoolExclude: &ex}); err != nil {
			t.Fatalf("UpdateSystemDataset: %v", err)
		}
	})

	t.Run("Update job FAILED propagates error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				_, _ = w.Write([]byte("7"))
				return
			}
			if strings.Contains(r.URL.Path, "/core/get_jobs") {
				writeJSON(w, http.StatusOK, []client.Job{{ID: 7, State: "FAILED", Error: "no pool"}})
			}
		}))
		pool := "badpool"
		_, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{Pool: &pool})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("Update empty body nils omitted", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "pool") {
				t.Errorf("body contains pool: %s", body)
			}
			writeJSON(w, http.StatusOK, client.SystemDataset{ID: 1})
		}))
		if _, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{}); err != nil {
			t.Fatalf("UpdateSystemDataset: %v", err)
		}
	})

	t.Run("Get path correctness", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/api/v2.0/systemdataset") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.SystemDataset{})
		}))
		_, _ = c.GetSystemDataset(ctx)
	})

	t.Run("Update path correctness", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/api/v2.0/systemdataset") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.SystemDataset{})
		}))
		_, _ = c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{})
	})

	t.Run("Get decodes all fields", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":2,"pool":"ssd","pool_set":true,"uuid":"abc","basename":"ssd/.system","path":"/p"}`))
		}))
		got, err := c.GetSystemDataset(ctx)
		if err != nil {
			t.Fatalf("GetSystemDataset: %v", err)
		}
		if got.ID != 2 || got.Pool != "ssd" || got.Basename != "ssd/.system" || got.Path != "/p" {
			t.Errorf("decoded = %+v", got)
		}
	})

	t.Run("Update unparseable returns error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`"not a job id or config"`))
		}))
		p := "x"
		_, err := c.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{Pool: &p})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Get empty pool valid", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.SystemDataset{ID: 1, Pool: "", PoolSet: false})
		}))
		got, err := c.GetSystemDataset(ctx)
		if err != nil {
			t.Fatalf("GetSystemDataset: %v", err)
		}
		if got.Pool != "" || got.PoolSet {
			t.Errorf("want empty/false, got %+v", got)
		}
	})
}
