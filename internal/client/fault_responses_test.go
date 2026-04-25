package client_test

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// Fault injection tests for malformed / truncated / slow responses from the
// TrueNAS API. Complements retry_test.go (which covers retryable status
// codes and transport errors) by exercising paths where the server returns
// a nominally-successful status but the body itself is broken.
//
// Every test asserts one of two properties:
//
//   - The client MUST surface the fault as an error (never silently accept
//     partial/garbage data).
//   - The client MUST NOT hang forever — context cancellation and HTTP
//     read deadlines always take effect.

// TestFault_InvalidJSONOnTypedMethod verifies that when the server returns
// HTTP 200 with a body that is not valid JSON, a typed high-level method
// (GetDataset) surfaces an unmarshal error instead of returning zero values.
func TestFault_InvalidJSONOnTypedMethod(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":`)) // truncated at the second character
	})
	_, c := newTestServer(t, handler)

	_, err := c.GetDataset(context.Background(), "tank/acct-fault")
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected") &&
		!strings.Contains(err.Error(), "EOF") &&
		!strings.Contains(err.Error(), "unmarshal") &&
		!strings.Contains(err.Error(), "JSON") &&
		!strings.Contains(err.Error(), "json") {
		t.Errorf("error does not look like a JSON parse failure: %v", err)
	}
}

// TestFault_WrongShape verifies that an object returned where an array was
// expected is caught by json.Unmarshal rather than silently decoded.
func TestFault_WrongShape(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"not":"an array"}`))
	})
	_, c := newTestServer(t, handler)

	// ListDatasets expects a JSON array.
	_, err := c.ListDatasets(context.Background())
	if err == nil {
		t.Fatal("expected error unmarshaling object as array, got nil")
	}
}

// TestFault_EmptyBodyOnTypedMethod verifies that an empty 200 body for a
// method that expects a JSON object surfaces as an error.
func TestFault_EmptyBodyOnTypedMethod(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// intentionally write nothing
	})
	_, c := newTestServer(t, handler)

	_, err := c.GetDataset(context.Background(), "tank/acct-empty")
	if err == nil {
		t.Fatal("expected error on empty body, got nil")
	}
}

// TestFault_SlowBody verifies that a server which hangs after writing the
// status line but before sending the body does not deadlock the client
// beyond the request context deadline.
func TestFault_SlowBody(t *testing.T) {
	block := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// flush so the client starts reading the body, then block
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-block
	})
	_, c := newTestServer(t, handler)
	// Cleanup order matters: newTestServer already registered srv.Close,
	// and t.Cleanup is LIFO, so this close(block) runs BEFORE srv.Close,
	// unblocking the handler so srv.Close can drain its active connections
	// without deadlocking on the Wait inside httptest.Server.Close.
	t.Cleanup(func() { close(block) })

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.Get(ctx, "/slow")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error (deadline exceeded), got nil")
	}
	if elapsed > 5*time.Second {
		t.Errorf("client hung past context deadline: elapsed=%s", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) &&
		!strings.Contains(err.Error(), "deadline") &&
		!strings.Contains(err.Error(), "canceled") &&
		!strings.Contains(err.Error(), "EOF") {
		t.Errorf("unexpected error shape: %v", err)
	}
}

// TestFault_ConnectionResetMidBody verifies that a server which hijacks the
// connection and closes it mid-body surfaces as a transport error (retryable
// for idempotent methods via the existing retry loop).
func TestFault_ConnectionResetMidBody(t *testing.T) {
	var callCount int
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		// On the first call, hijack and close to simulate a RST.
		if callCount == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("response writer does not support Hijacker")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 1000\r\n\r\n{\"id\":\"tank/acct-rst\",\"type\"")) // short body
			_ = conn.Close()
			return
		}
		// Subsequent calls succeed.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"tank/acct-rst","type":"FILESYSTEM"}`))
	})
	_, c := newTestServer(t, handler)

	// GetDataset goes through a GET, which is idempotent and retryable.
	ds, err := c.GetDataset(context.Background(), "tank/acct-rst")
	if err != nil {
		t.Fatalf("expected retry to succeed after RST: %v", err)
	}
	if ds == nil || ds.ID != "tank/acct-rst" {
		t.Errorf("retry returned unexpected dataset: %+v", ds)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 calls (1 reset + 1 retry), got %d", callCount)
	}
}

// TestFault_RawSocketGarbage verifies that a server responding with bytes
// that are not a valid HTTP response surfaces as a transport error rather
// than a panic or hang. We use net.Listen directly because httptest.Server
// always writes a valid HTTP response.
func TestFault_RawSocketGarbage(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Consume the request line so Accept doesn't just sit.
			_, _ = bufio.NewReader(conn).ReadString('\n')
			_, _ = conn.Write([]byte("garbage\x00\x01\x02 not http\n"))
			_ = conn.Close()
		}
	}()

	// Build a client pointing at our garbage server. Use http:// scheme
	// so the client does not try to TLS-handshake.
	url := "http://" + ln.Addr().String()
	c, err := client.New(url, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.Get(ctx, "/anything")
	if err == nil {
		t.Fatal("expected transport error on garbage response, got nil")
	}
}
