package client_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestReadOnly_TypedCRUDMethods verifies that the read-only safety rail
// holds through the high-level typed CRUD methods, not just through the
// raw Post/Put/Delete primitives. This is the plumbing test that proves
// no CreateXxx / UpdateXxx / DeleteXxx method accidentally catches and
// swallows ErrReadOnly on its way up the stack.
//
// One representative per verb is enough — every typed method ultimately
// flows through client.doRequest, which is where the gate lives, so if
// the three-of-a-kind below works the invariant holds for the entire
// client surface. Adding a new typed method without affecting the
// invariant is safe because the gate is applied to the underlying HTTP
// method, not to the typed wrapper.
func TestReadOnly_TypedCRUDMethods(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	_, c := newTestServer(t, handler)
	c.ReadOnly = true

	t.Run("CreateDataset_blocked", func(t *testing.T) {
		_, err := c.CreateDataset(context.Background(), &client.DatasetCreateRequest{
			Name: "tank/acct-readonly",
			Type: "FILESYSTEM",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, client.ErrReadOnly) {
			t.Errorf("want ErrReadOnly, got %v", err)
		}
	})

	t.Run("UpdateDataset_blocked", func(t *testing.T) {
		_, err := c.UpdateDataset(context.Background(), "tank/acct-readonly",
			&client.DatasetUpdateRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, client.ErrReadOnly) {
			t.Errorf("want ErrReadOnly, got %v", err)
		}
	})

	t.Run("DeleteDataset_blocked", func(t *testing.T) {
		err := c.DeleteDataset(context.Background(), "tank/acct-readonly")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, client.ErrReadOnly) {
			t.Errorf("want ErrReadOnly, got %v", err)
		}
	})

	// The server MUST NOT have received any requests at all — the gate
	// fires before any HTTP work, so the handler's call counter stays 0.
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("server received %d calls, want 0 — read-only gate leaked a request", got)
	}

	// Reads MUST still work. This is the invariant that makes read-only
	// mode operationally useful: `terraform plan` depends on GetDataset /
	// ListDatasets returning live data.
	t.Run("GetDataset_allowed", func(t *testing.T) {
		// A 200 OK with empty body will fail at the JSON unmarshal step
		// (fault_responses_test covers that), but crucially it will NOT
		// fail with ErrReadOnly — which is all we care about here.
		_, err := c.GetDataset(context.Background(), "tank/acct-readonly")
		if errors.Is(err, client.ErrReadOnly) {
			t.Errorf("GetDataset failed with ErrReadOnly — the gate is over-eager: %v", err)
		}
	})
}
