package client_test

// newTestServer and writeJSON helpers are defined in client_test.go (same
// package). go vet at package level resolves them; single-file vet does not.

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

func TestGetTunable_Success(t *testing.T) {
	ctx := context.Background()
	want := client.Tunable{
		ID: 1, Type: "SYSCTL", Var: "net.inet.tcp.mssdflt",
		Value: "1460", Comment: "test", Enabled: true,
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/tunable/id/1") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetTunable(ctx, 1)
	if err != nil {
		t.Fatalf("GetTunable: %v", err)
	}
	if got.Var != "net.inet.tcp.mssdflt" {
		t.Errorf("Var: %q", got.Var)
	}
	if got.Value != "1460" {
		t.Errorf("Value: %q", got.Value)
	}
}

func TestGetTunable_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "missing"})
	}))
	_, err := c.GetTunable(ctx, 999)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetTunable_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xx")
	}))
	_, err := c.GetTunable(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing tunable") {
		t.Errorf("got: %v", err)
	}
}

func TestGetTunable_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.GetTunable(ctx, 1)
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

func TestListTunables_Success(t *testing.T) {
	ctx := context.Background()
	want := []client.Tunable{
		{ID: 1, Type: "SYSCTL", Var: "a", Value: "1", Enabled: true},
		{ID: 2, Type: "SYSCTL", Var: "b", Value: "2", Enabled: false},
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/tunable") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.ListTunables(ctx)
	if err != nil {
		t.Fatalf("ListTunables: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len: %d", len(got))
	}
	if got[0].Var != "a" || got[1].Var != "b" {
		t.Errorf("vars: %+v", got)
	}
}

func TestListTunables_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xx")
	}))
	_, err := c.ListTunables(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing tunables list") {
		t.Errorf("got: %v", err)
	}
}

func TestListTunables_Empty(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Tunable{})
	}))
	got, err := c.ListTunables(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty: %+v", got)
	}
}

func TestCreateTunable_Success(t *testing.T) {
	// CreateTunable calls POST /tunable then ListTunables and finds by Var name.
	ctx := context.Background()
	call := 0
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// POST /tunable
			if r.Method != http.MethodPost {
				t.Errorf("first call method: %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.TunableCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Var != "vm.swappiness" {
				t.Errorf("Var: %q", req.Var)
			}
			if req.Type != "SYSCTL" {
				t.Errorf("Type: %q", req.Type)
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"id": 999})
			return
		}
		// GET /tunable list
		if r.Method != http.MethodGet {
			t.Errorf("second call method: %s", r.Method)
		}
		writeJSON(w, http.StatusOK, []client.Tunable{
			{ID: 1, Var: "other", Value: "x"},
			{ID: 2, Var: "vm.swappiness", Value: "10", Type: "SYSCTL", Enabled: true},
		})
	}))

	got, err := c.CreateTunable(ctx, &client.TunableCreateRequest{
		Type: "SYSCTL", Var: "vm.swappiness", Value: "10", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateTunable: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("ID: %d, want 2 (found by var)", got.ID)
	}
}

func TestCreateTunable_NotFoundAfterCreate(t *testing.T) {
	ctx := context.Background()
	call := 0
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			writeJSON(w, http.StatusOK, map[string]interface{}{"id": 1})
			return
		}
		writeJSON(w, http.StatusOK, []client.Tunable{})
	}))

	_, err := c.CreateTunable(ctx, &client.TunableCreateRequest{
		Type: "SYSCTL", Var: "nonexistent", Value: "1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found after creation") {
		t.Errorf("got: %v", err)
	}
}

func TestCreateTunable_PostError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid"})
	}))
	_, err := c.CreateTunable(ctx, &client.TunableCreateRequest{
		Type: "SYSCTL", Var: "x", Value: "1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "invalid" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestFindTunableByVar_Found(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Tunable{
			{ID: 1, Var: "a"},
			{ID: 2, Var: "b"},
		})
	}))
	got, err := c.FindTunableByVar(ctx, "b")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("ID: %d", got.ID)
	}
}

func TestFindTunableByVar_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Tunable{})
	}))
	_, err := c.FindTunableByVar(ctx, "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateTunable_Success(t *testing.T) {
	// UpdateTunable calls PUT then re-fetches via GetTunable.
	ctx := context.Background()
	call := 0
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			if r.Method != http.MethodPut {
				t.Errorf("first call method: %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tunable/id/5") {
				t.Errorf("path: %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.TunableUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Value != "20" {
				t.Errorf("Value: %q", req.Value)
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"id": 999})
			return
		}
		// GET /tunable/id/5
		if r.Method != http.MethodGet {
			t.Errorf("second call method: %s", r.Method)
		}
		writeJSON(w, http.StatusOK, client.Tunable{ID: 5, Var: "vm.swappiness", Value: "20"})
	}))

	enabled := true
	got, err := c.UpdateTunable(ctx, 5, &client.TunableUpdateRequest{
		Value: "20", Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("UpdateTunable: %v", err)
	}
	if got.ID != 5 || got.Value != "20" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestUpdateTunable_PutError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateTunable(ctx, 1, &client.TunableUpdateRequest{Value: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "updating tunable") {
		t.Errorf("got: %v", err)
	}
}

func TestDeleteTunable_Success(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/tunable/id/7") {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.DeleteTunable(ctx, 7); err != nil {
		t.Fatalf("DeleteTunable: %v", err)
	}
}

func TestDeleteTunable_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "missing"})
	}))
	err := c.DeleteTunable(ctx, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestDeleteTunable_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "locked"})
	}))
	err := c.DeleteTunable(ctx, 1)
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

func TestGetTunable_URLFormat(t *testing.T) {
	ctx := context.Background()
	var gotPath string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, client.Tunable{ID: 42})
	}))
	if _, err := c.GetTunable(ctx, 42); err != nil {
		t.Fatal(err)
	}
	want := "/tunable/id/42"
	if !strings.HasSuffix(gotPath, want) {
		t.Errorf("path: got %s, want suffix %s", gotPath, want)
	}
}

func TestFindTunableByVar_ListError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "err"})
	}))
	_, err := c.FindTunableByVar(ctx, "x")
	if err == nil {
		t.Fatal("expected error")
	}
}
