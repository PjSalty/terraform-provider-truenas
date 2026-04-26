package client

// Benchmarks for hot paths in the TrueNAS HTTP client. These are package-
// internal so they can exercise unexported helpers directly. Run with:
//
//	go test -run='^$' -bench=. -benchtime=5s ./internal/client/...
//
// Use -benchtime=1x in CI to keep pipelines fast.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkDoRequest_Success benchmarks the happy-path request cycle:
// build request → send → decode response. No retries, no errors. This is
// the dominant cost for every CRUD op in every resource.
func BenchmarkDoRequest_Success(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    1,
			"name":  "bench",
			"state": "RUNNING",
		})
	}))
	defer srv.Close()

	c, err := New(srv.URL, "bench-key")
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.Get(ctx, "/bench"); err != nil {
			b.Fatalf("Get: %v", err)
		}
	}
}

// BenchmarkBackoffDelay benchmarks the tight-loop arithmetic for computing
// exponential-backoff delays. Called once per retry attempt but must be
// cheap enough to not dominate the retry cost when delays are small.
func BenchmarkBackoffDelay(b *testing.B) {
	policy := RetryPolicy{BaseDelay: 100 * time.Millisecond, MaxDelay: 30 * time.Second}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backoffDelay(policy, i%8)
	}
}

// BenchmarkReadOnlyOverhead compares the cost of a successful GET with
// ReadOnly disabled vs enabled. The safety rail should add zero overhead
// to read paths — if this benchmark shows a non-trivial delta, the gate
// grew an allocation or a lock we did not intend. Run with:
//
//	go test -run='^$' -bench=BenchmarkReadOnlyOverhead -benchtime=3s ./internal/client/
//
// Expected: OFF and ON nanoseconds-per-op should match to within noise,
// because the gate check is a single bool test followed by a method
// switch for non-safe verbs, and GET is the early-return path.
func BenchmarkReadOnlyOverhead(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer srv.Close()

	c, err := New(srv.URL, "bench-key")
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	b.Run("OFF", func(b *testing.B) {
		c.ReadOnly = false
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := c.Get(ctx, "/bench"); err != nil {
				b.Fatalf("Get: %v", err)
			}
		}
	})

	b.Run("ON", func(b *testing.B) {
		c.ReadOnly = true
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := c.Get(ctx, "/bench"); err != nil {
				b.Fatalf("Get: %v", err)
			}
		}
	})
}

// BenchmarkReadOnlyGate_BlockedRequest benchmarks the cost of the gate
// firing on a mutating request. This is the path that returns without
// doing any network work at all, so it should be effectively free.
// Measuring it proves the gate is not accidentally allocating on the
// rejection path (formatting the error message, wrapping, etc).
func BenchmarkReadOnlyGate_BlockedRequest(b *testing.B) {
	c, err := New("http://example.invalid", "bench-key")
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	c.ReadOnly = true
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Delete on a readonly client short-circuits in checkReadOnly
		// before any HTTP work. We expect the error, we do not expect
		// the HTTP client.invalid host to ever be contacted.
		_, _ = c.Delete(ctx, "/bench")
	}
}
