package client_test

import (
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestRequestTimeout_Default verifies the default timeout is 60s.
func TestRequestTimeout_Default(t *testing.T) {
	c, err := client.New("http://example.invalid", "k")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.RequestTimeout(); got != 60*time.Second {
		t.Errorf("default RequestTimeout = %s, want 60s", got)
	}
	if client.DefaultRequestTimeout != 60*time.Second {
		t.Errorf("DefaultRequestTimeout constant = %s, want 60s", client.DefaultRequestTimeout)
	}
}

// TestRequestTimeout_Set verifies SetRequestTimeout updates the client.
func TestRequestTimeout_Set(t *testing.T) {
	c, err := client.New("http://example.invalid", "k")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.SetRequestTimeout(5 * time.Minute)
	if got := c.RequestTimeout(); got != 5*time.Minute {
		t.Errorf("RequestTimeout after Set = %s, want 5m", got)
	}
}

// TestRequestTimeout_ZeroIgnored verifies that a zero duration does not
// disable the timeout. The safety rail matters — a misconfigured env var
// should never silently remove the per-request deadline.
func TestRequestTimeout_ZeroIgnored(t *testing.T) {
	c, err := client.New("http://example.invalid", "k")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	original := c.RequestTimeout()
	c.SetRequestTimeout(0)
	if got := c.RequestTimeout(); got != original {
		t.Errorf("SetRequestTimeout(0) changed timeout to %s, want original %s", got, original)
	}
}

// TestRequestTimeout_NegativeIgnored is the sibling of the zero test —
// also guards against a negative value disabling the rail.
func TestRequestTimeout_NegativeIgnored(t *testing.T) {
	c, err := client.New("http://example.invalid", "k")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	original := c.RequestTimeout()
	c.SetRequestTimeout(-30 * time.Second)
	if got := c.RequestTimeout(); got != original {
		t.Errorf("SetRequestTimeout(-30s) changed timeout to %s, want original %s", got, original)
	}
}

// TestRequestTimeout_NilClientSafe exercises the nil-receiver guards so
// callers can safely invoke the setter/getter on a zero value.
func TestRequestTimeout_NilClientSafe(t *testing.T) {
	var c *client.Client
	c.SetRequestTimeout(10 * time.Second) // must not panic
	if got := c.RequestTimeout(); got != 0 {
		t.Errorf("nil client RequestTimeout() = %s, want 0", got)
	}
}
