package wsclient

import (
	"errors"
	"testing"
)

func TestIsDestructiveMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"", false},
		{"pool.dataset.delete", true},
		{"pool.dataset.destroy", true},
		{"share.smb.remove", true},
		{"system.unset", true},
		{"foo.purge", true},
		{"foo.wipe", true},
		{"pool.dataset.create", false},
		{"pool.dataset.update", false},
		{"pool.dataset.query", false},
		{"system.info", false},
		{"delete", true}, // single segment
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := isDestructiveMethod(tt.method)
			if got != tt.want {
				t.Errorf("isDestructiveMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestCheckDestroyProtection(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		var c *Client
		if err := c.checkDestroyProtection("pool.dataset.delete", CallOptions{}); err != nil {
			t.Errorf("nil client should return nil, got %v", err)
		}
	})

	t.Run("DestroyProtection false passes destructive", func(t *testing.T) {
		c := &Client{DestroyProtection: false}
		if err := c.checkDestroyProtection("pool.dataset.delete", CallOptions{}); err != nil {
			t.Errorf("DestroyProtection=false should allow delete, got %v", err)
		}
	})

	t.Run("DestroyProtection true blocks destructive", func(t *testing.T) {
		c := &Client{DestroyProtection: true}
		err := c.checkDestroyProtection("pool.dataset.delete", CallOptions{})
		if !errors.Is(err, ErrDestroyProtected) {
			t.Errorf("expected ErrDestroyProtected, got %v", err)
		}
	})

	t.Run("DestroyProtection true passes non-destructive", func(t *testing.T) {
		c := &Client{DestroyProtection: true}
		if err := c.checkDestroyProtection("pool.dataset.create", CallOptions{}); err != nil {
			t.Errorf("DestroyProtection=true should allow create, got %v", err)
		}
		if err := c.checkDestroyProtection("pool.dataset.update", CallOptions{}); err != nil {
			t.Errorf("DestroyProtection=true should allow update, got %v", err)
		}
	})

	t.Run("Destroys override forces block", func(t *testing.T) {
		c := &Client{DestroyProtection: true}
		err := c.checkDestroyProtection("system.reboot", CallOptions{Destroys: true})
		if !errors.Is(err, ErrDestroyProtected) {
			t.Errorf("Destroys=true should force block, got %v", err)
		}
	})
}
