package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestISCSITarget_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.ISCSITarget{
				ID:   2,
				Name: "iqn.2005-10.org.freenas.ctl:target1",
				Mode: "ISCSI",
				Groups: []client.ISCSITargetGroup{
					{Portal: 1, Initiator: 1, AuthMethod: "NONE"},
				},
			})
		}))

		got, err := c.GetISCSITarget(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "iqn.2005-10.org.freenas.ctl:target1" {
			t.Errorf("Name = %q", got.Name)
		}
		if len(got.Groups) != 1 || got.Groups[0].Portal != 1 {
			t.Errorf("Groups = %+v", got.Groups)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))

		_, err := c.GetISCSITarget(ctx, 99)
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
			var req client.ISCSITargetCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Mode != "ISCSI" {
				t.Errorf("Mode = %q", req.Mode)
			}
			if len(req.Groups) != 1 {
				t.Errorf("Groups len = %d", len(req.Groups))
			}
			writeJSON(w, http.StatusOK, client.ISCSITarget{
				ID:     11,
				Name:   req.Name,
				Mode:   req.Mode,
				Groups: req.Groups,
			})
		}))

		resp, err := c.CreateISCSITarget(ctx, &client.ISCSITargetCreateRequest{
			Name: "iqn.example.com:lun0",
			Mode: "ISCSI",
			Groups: []client.ISCSITargetGroup{
				{Portal: 1, Initiator: 1, AuthMethod: "NONE", Auth: 0},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 11 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.ISCSITarget{
				ID:    11,
				Name:  "iqn.example.com:lun0",
				Alias: "renamed",
				Mode:  "ISCSI",
			})
		}))

		resp, err := c.UpdateISCSITarget(ctx, 11, &client.ISCSITargetUpdateRequest{
			Alias: "renamed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Alias != "renamed" {
			t.Errorf("Alias = %q", resp.Alias)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteISCSITarget(ctx, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
