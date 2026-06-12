package wsclient

import (
	"errors"
	"testing"
)

func TestIsReadOnlyMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"", false},
		{"system.info", true},
		{"pool.dataset.query", true},
		{"pool.dataset.get_instance", true},
		{"system.version", true},
		{"core.ping", true},
		{"sharing.nfs.choices", true},
		{"core.get", true},
		{"pool.dataset.create", false},
		{"pool.dataset.update", false},
		{"pool.dataset.delete", false},
		{"pool.dataset.lock", false},
		{"unknownmethod", false},
		{"info", true}, // single-segment match
		{"check", true},
		{"list", true},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := isReadOnlyMethod(tt.method)
			if got != tt.want {
				t.Errorf("isReadOnlyMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestCheckReadOnly(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		var c *Client
		if err := c.checkReadOnly("pool.dataset.create", CallOptions{}); err != nil {
			t.Errorf("nil client should return nil, got %v", err)
		}
	})

	t.Run("ReadOnly false passes mutating", func(t *testing.T) {
		c := &Client{ReadOnly: false}
		if err := c.checkReadOnly("pool.dataset.create", CallOptions{}); err != nil {
			t.Errorf("ReadOnly=false should allow mutating, got %v", err)
		}
	})

	t.Run("ReadOnly true blocks mutating", func(t *testing.T) {
		c := &Client{ReadOnly: true}
		err := c.checkReadOnly("pool.dataset.create", CallOptions{})
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("expected ErrReadOnly, got %v", err)
		}
	})

	t.Run("ReadOnly true passes safe read", func(t *testing.T) {
		c := &Client{ReadOnly: true}
		if err := c.checkReadOnly("system.info", CallOptions{}); err != nil {
			t.Errorf("ReadOnly=true should allow read, got %v", err)
		}
	})

	t.Run("Mutates override blocks even read-suffixed method", func(t *testing.T) {
		c := &Client{ReadOnly: true}
		err := c.checkReadOnly("system.info", CallOptions{Mutates: true})
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("Mutates=true should force block, got %v", err)
		}
	})

	t.Run("Read override allows mutating-suffixed method", func(t *testing.T) {
		c := &Client{ReadOnly: true}
		if err := c.checkReadOnly("auth.login_with_api_key", CallOptions{Read: true}); err != nil {
			t.Errorf("Read=true should bypass classifier, got %v", err)
		}
	})
}
