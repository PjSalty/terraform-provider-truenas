package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestISCSIExtent_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get DISK type", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Use a raw JSON to model what the API actually returns:
			// disk is a string, filesize is a number (or string).
			_, _ = w.Write([]byte(`{
				"id": 3,
				"name": "lun0",
				"type": "DISK",
				"disk": "zvol/tank/lun0",
				"filesize": 0,
				"blocksize": 512,
				"enabled": true,
				"ro": false,
				"xen": false,
				"insecure_tpc": true
			}`))
		}))

		got, err := c.GetISCSIExtent(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "lun0" {
			t.Errorf("Name = %q", got.Name)
		}
		if got.GetDisk() != "zvol/tank/lun0" {
			t.Errorf("GetDisk() = %q", got.GetDisk())
		}
		if got.GetFilesize() != 0 {
			t.Errorf("GetFilesize() = %d", got.GetFilesize())
		}
	})

	t.Run("Get FILE type with string filesize", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 4,
				"name": "file-lun",
				"type": "FILE",
				"disk": null,
				"path": "/mnt/tank/lun.img",
				"filesize": "1073741824",
				"blocksize": 4096,
				"enabled": true
			}`))
		}))

		got, err := c.GetISCSIExtent(ctx, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.GetDisk() != "" {
			t.Errorf("GetDisk() = %q, want empty for null", got.GetDisk())
		}
		if got.GetFilesize() != 1073741824 {
			t.Errorf("GetFilesize() = %d, want 1073741824", got.GetFilesize())
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))

		_, err := c.GetISCSIExtent(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			var req client.ISCSIExtentCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "new-lun" {
				t.Errorf("Name = %q", req.Name)
			}
			if req.Blocksize != 512 {
				t.Errorf("Blocksize = %d", req.Blocksize)
			}
			_, _ = w.Write([]byte(`{
				"id": 8,
				"name": "new-lun",
				"type": "DISK",
				"disk": "zvol/tank/new-lun",
				"blocksize": 512,
				"enabled": true
			}`))
		}))

		resp, err := c.CreateISCSIExtent(ctx, &client.ISCSIExtentCreateRequest{
			Name:      "new-lun",
			Type:      "DISK",
			Disk:      "zvol/tank/new-lun",
			Blocksize: 512,
			Enabled:   true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 8 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"id": 8, "name": "renamed-lun", "type": "DISK", "blocksize": 512, "enabled": false}`))
		}))

		enabled := false
		resp, err := c.UpdateISCSIExtent(ctx, 8, &client.ISCSIExtentUpdateRequest{
			Name:    "renamed-lun",
			Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "renamed-lun" {
			t.Errorf("Name = %q", resp.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteISCSIExtent(ctx, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestISCSIExtent_GetDiskNull(t *testing.T) {
	e := client.ISCSIExtent{Disk: nil}
	if e.GetDisk() != "" {
		t.Errorf("GetDisk nil Disk = %q, want empty", e.GetDisk())
	}
}

func TestISCSIExtent_GetFilesizeEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int64
	}{
		{"empty", ``, 0},
		{"null", `null`, 0},
		{"number", `42`, 42},
		{"string", `"42"`, 42},
		{"invalid", `"abc"`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := client.ISCSIExtent{}
			if tc.raw != "" {
				e.Filesize = []byte(tc.raw)
			}
			if got := e.GetFilesize(); got != tc.want {
				t.Errorf("GetFilesize(%s) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}
