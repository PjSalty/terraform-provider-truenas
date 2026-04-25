// Tests for scrub_task CRUD.
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

func TestScrubTask_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.ScrubTask{
				ID:        1,
				Pool:      1,
				PoolName:  "tank",
				Threshold: 35,
				Enabled:   true,
				Schedule: client.Schedule{
					Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "0",
				},
			})
		}))

		got, err := c.GetScrubTask(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.PoolName != "tank" {
			t.Errorf("PoolName = %q", got.PoolName)
		}
		if got.Threshold != 35 {
			t.Errorf("Threshold = %d", got.Threshold)
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.ScrubTask{
				ID:      3,
				Pool:    1,
				Enabled: true,
			})
		}))

		resp, err := c.CreateScrubTask(ctx, &client.ScrubTaskCreateRequest{
			Pool:      1,
			Threshold: 7,
			Schedule: client.Schedule{
				Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "0",
			},
			Enabled: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 3 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.ScrubTask{
				ID:        3,
				Threshold: 14,
				Enabled:   false,
			})
		}))

		enabled := false
		resp, err := c.UpdateScrubTask(ctx, 3, &client.ScrubTaskUpdateRequest{
			Threshold: 14,
			Enabled:   &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Threshold != 14 {
			t.Errorf("Threshold = %d", resp.Threshold)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteScrubTask(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGetScrubTask_URLFormat(t *testing.T) {
	ctx := context.Background()
	var gotPath string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		writeJSON(w, http.StatusOK, client.ScrubTask{ID: 42})
	}))
	if _, err := c.GetScrubTask(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(gotPath, "/pool/scrub/id/42") {
		t.Errorf("path: %s", gotPath)
	}
}

func TestGetScrubTask_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetScrubTask(ctx, 99)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetScrubTask_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xx")
	}))
	_, err := c.GetScrubTask(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing scrub task") {
		t.Errorf("got: %v", err)
	}
}

func TestGetScrubTask_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.GetScrubTask(ctx, 1)
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

func TestCreateScrubTask_RequestBody(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.ScrubTaskCreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Pool != 1 || req.Threshold != 35 {
			t.Errorf("body wrong: %+v", req)
		}
		if req.Schedule.Hour != "0" {
			t.Errorf("Schedule.Hour: %q", req.Schedule.Hour)
		}
		writeJSON(w, http.StatusOK, client.ScrubTask{ID: 10, Pool: 1, Threshold: 35})
	}))

	_, err := c.CreateScrubTask(ctx, &client.ScrubTaskCreateRequest{
		Pool: 1, Threshold: 35,
		Schedule: client.Schedule{Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "0"},
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestCreateScrubTask_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid pool"})
	}))
	_, err := c.CreateScrubTask(ctx, &client.ScrubTaskCreateRequest{Pool: 999})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "invalid pool" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestCreateScrubTask_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "bogus")
	}))
	_, err := c.CreateScrubTask(ctx, &client.ScrubTaskCreateRequest{Pool: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing scrub task create") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateScrubTask_URLFormat(t *testing.T) {
	ctx := context.Background()
	var gotPath string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		writeJSON(w, http.StatusOK, client.ScrubTask{ID: 7})
	}))
	if _, err := c.UpdateScrubTask(ctx, 7, &client.ScrubTaskUpdateRequest{}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(gotPath, "/pool/scrub/id/7") {
		t.Errorf("path: %s", gotPath)
	}
}

func TestUpdateScrubTask_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateScrubTask(ctx, 1, &client.ScrubTaskUpdateRequest{})
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

func TestUpdateScrubTask_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "garbage")
	}))
	_, err := c.UpdateScrubTask(ctx, 1, &client.ScrubTaskUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing scrub task update") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateScrubTask_WithSchedule(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "schedule") {
			t.Errorf("missing schedule: %s", body)
		}
		writeJSON(w, http.StatusOK, client.ScrubTask{ID: 1})
	}))
	sched := client.Schedule{Minute: "30", Hour: "2", Dom: "*", Month: "*", Dow: "*"}
	_, err := c.UpdateScrubTask(ctx, 1, &client.ScrubTaskUpdateRequest{Schedule: &sched})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestDeleteScrubTask_URLFormat(t *testing.T) {
	ctx := context.Background()
	var gotPath string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodDelete {
			t.Errorf("method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.DeleteScrubTask(ctx, 11); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(gotPath, "/pool/scrub/id/11") {
		t.Errorf("path: %s", gotPath)
	}
}

func TestDeleteScrubTask_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
	}))
	err := c.DeleteScrubTask(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestDeleteScrubTask_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "locked"})
	}))
	err := c.DeleteScrubTask(ctx, 1)
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
