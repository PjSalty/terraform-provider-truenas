package client_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNVMet_Global(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetGlobal{
				ID:      1,
				Basenqn: "nqn.2011-06.com.truenas",
				Kernel:  true,
				Ana:     false,
			})
		}))

		got, err := c.GetNVMetGlobal(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Basenqn != "nqn.2011-06.com.truenas" {
			t.Errorf("Basenqn = %q", got.Basenqn)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusOK, client.NVMetGlobal{
				ID:      1,
				Basenqn: "nqn.2011-06.com.truenas",
				Kernel:  true,
				Ana:     true,
			})
		}))

		ana := true
		resp, err := c.UpdateNVMetGlobal(ctx, &client.NVMetGlobalUpdateRequest{
			Ana: &ana,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Ana {
			t.Errorf("Ana = false, want true")
		}
	})

	t.Run("Get error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))

		_, err := c.GetNVMetGlobal(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNVMet_Host(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetHost{
				ID:      1,
				Hostnqn: "nqn.2014-08.org.nvmexpress:uuid:host1",
			})
		}))

		got, err := c.GetNVMetHost(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Hostnqn != "nqn.2014-08.org.nvmexpress:uuid:host1" {
			t.Errorf("Hostnqn = %q", got.Hostnqn)
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetHost{
				ID:      7,
				Hostnqn: "nqn.2014-08.org.nvmexpress:uuid:newhost",
			})
		}))

		resp, err := c.CreateNVMetHost(ctx, &client.NVMetHostCreateRequest{
			Hostnqn: "nqn.2014-08.org.nvmexpress:uuid:newhost",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 7 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetHost{ID: 7, Hostnqn: "nqn.updated"})
		}))

		nqn := "nqn.updated"
		resp, err := c.UpdateNVMetHost(ctx, 7, &client.NVMetHostUpdateRequest{Hostnqn: &nqn})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Hostnqn != "nqn.updated" {
			t.Errorf("Hostnqn = %q", resp.Hostnqn)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNVMetHost(ctx, 7); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNVMet_Subsys(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetSubsys{
				ID:           3,
				Name:         "subsys1",
				AllowAnyHost: true,
			})
		}))

		got, err := c.GetNVMetSubsys(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "subsys1" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetSubsys{ID: 5, Name: "new"})
		}))

		allow := true
		resp, err := c.CreateNVMetSubsys(ctx, &client.NVMetSubsysCreateRequest{
			Name:         "new",
			AllowAnyHost: &allow,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 5 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.NVMetSubsys{ID: 5, Name: "renamed"})
		}))

		name := "renamed"
		_, err := c.UpdateNVMetSubsys(ctx, 5, &client.NVMetSubsysUpdateRequest{Name: &name})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNVMetSubsys(ctx, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNVMet_Port(t *testing.T) {
	ctx := context.Background()

	t.Run("Get with int trsvcid", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 1,
				"index": 0,
				"addr_trtype": "TCP",
				"addr_trsvcid": 4420,
				"addr_traddr": "192.0.2.10",
				"enabled": true
			}`))
		}))

		got, err := c.GetNVMetPort(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.AddrTrtype != "TCP" {
			t.Errorf("AddrTrtype = %q", got.AddrTrtype)
		}
		if got.GetAddrTrsvcid() != 4420 {
			t.Errorf("GetAddrTrsvcid() = %d", got.GetAddrTrsvcid())
		}
	})

	t.Run("Get with string trsvcid", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 2,
				"index": 1,
				"addr_trtype": "RDMA",
				"addr_trsvcid": "4420",
				"addr_traddr": "192.0.2.11",
				"enabled": true
			}`))
		}))

		got, err := c.GetNVMetPort(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.GetAddrTrsvcid() != 4420 {
			t.Errorf("GetAddrTrsvcid() = %d", got.GetAddrTrsvcid())
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 4,
				"index": 0,
				"addr_trtype": "TCP",
				"addr_trsvcid": 4420,
				"addr_traddr": "192.0.2.10",
				"enabled": true
			}`))
		}))

		port := 4420
		enabled := true
		resp, err := c.CreateNVMetPort(ctx, &client.NVMetPortCreateRequest{
			AddrTrtype:  "TCP",
			AddrTraddr:  "192.0.2.10",
			AddrTrsvcid: &port,
			Enabled:     &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 4 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":4,"index":0,"addr_trtype":"TCP","addr_traddr":"192.0.2.10","enabled":false}`))
		}))

		enabled := false
		_, err := c.UpdateNVMetPort(ctx, 4, &client.NVMetPortUpdateRequest{Enabled: &enabled})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNVMetPort(ctx, 4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNVMetPort_GetAddrTrsvcidEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int
	}{
		{"empty", ``, 0},
		{"null", `null`, 0},
		{"int", `4420`, 4420},
		{"string", `"4420"`, 4420},
		{"invalid", `"abc"`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := client.NVMetPort{}
			if tc.raw != "" {
				p.AddrTrsvcid = []byte(tc.raw)
			}
			if got := p.GetAddrTrsvcid(); got != tc.want {
				t.Errorf("GetAddrTrsvcid() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestNVMet_Namespace(t *testing.T) {
	ctx := context.Background()

	t.Run("Get nested subsys form", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 1,
				"nsid": 1,
				"subsys": {"id": 3},
				"device_type": "ZVOL",
				"device_path": "zvol/tank/ns1",
				"enabled": true
			}`))
		}))

		got, err := c.GetNVMetNamespace(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.EffectiveSubsysID() != 3 {
			t.Errorf("EffectiveSubsysID() = %d, want 3", got.EffectiveSubsysID())
		}
	})

	t.Run("Get flat subsys_id form", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 2,
				"subsys_id": 5,
				"device_type": "ZVOL",
				"device_path": "zvol/tank/ns2",
				"enabled": true
			}`))
		}))

		got, err := c.GetNVMetNamespace(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.EffectiveSubsysID() != 5 {
			t.Errorf("EffectiveSubsysID() = %d, want 5", got.EffectiveSubsysID())
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 10,
				"subsys": {"id": 3},
				"device_type": "ZVOL",
				"device_path": "zvol/tank/new",
				"enabled": true
			}`))
		}))

		resp, err := c.CreateNVMetNamespace(ctx, &client.NVMetNamespaceCreateRequest{
			DeviceType: "ZVOL",
			DevicePath: "zvol/tank/new",
			SubsysID:   3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 10 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":10,"device_type":"ZVOL","device_path":"zvol/tank/new","enabled":false}`))
		}))

		enabled := false
		_, err := c.UpdateNVMetNamespace(ctx, 10, &client.NVMetNamespaceUpdateRequest{Enabled: &enabled})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNVMetNamespace(ctx, 10); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNVMet_HostSubsys(t *testing.T) {
	ctx := context.Background()

	t.Run("Get nested", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 1,
				"host": {"id": 2},
				"subsys": {"id": 3}
			}`))
		}))

		got, err := c.GetNVMetHostSubsys(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.EffectiveHostID() != 2 {
			t.Errorf("EffectiveHostID() = %d", got.EffectiveHostID())
		}
		if got.EffectiveSubsysID() != 3 {
			t.Errorf("EffectiveSubsysID() = %d", got.EffectiveSubsysID())
		}
	})

	t.Run("Get flat", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":9,"host_id":4,"subsys_id":5}`))
		}))

		got, err := c.GetNVMetHostSubsys(ctx, 9)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.EffectiveHostID() != 4 {
			t.Errorf("EffectiveHostID() = %d", got.EffectiveHostID())
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":11,"host":{"id":2},"subsys":{"id":3}}`))
		}))

		resp, err := c.CreateNVMetHostSubsys(ctx, &client.NVMetHostSubsysCreateRequest{
			HostID:   2,
			SubsysID: 3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 11 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNVMetHostSubsys(ctx, 11); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNVMet_PortSubsys(t *testing.T) {
	ctx := context.Background()

	t.Run("Get nested", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":1,"port":{"id":4},"subsys":{"id":5}}`))
		}))

		got, err := c.GetNVMetPortSubsys(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.EffectivePortID() != 4 {
			t.Errorf("EffectivePortID() = %d", got.EffectivePortID())
		}
		if got.EffectiveSubsysID() != 5 {
			t.Errorf("EffectiveSubsysID() = %d", got.EffectiveSubsysID())
		}
	})

	t.Run("Get flat", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":2,"port_id":4,"subsys_id":5}`))
		}))

		got, err := c.GetNVMetPortSubsys(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.EffectivePortID() != 4 {
			t.Errorf("EffectivePortID() = %d", got.EffectivePortID())
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":12,"port":{"id":4},"subsys":{"id":5}}`))
		}))

		resp, err := c.CreateNVMetPortSubsys(ctx, &client.NVMetPortSubsysCreateRequest{
			PortID:   4,
			SubsysID: 5,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 12 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNVMetPortSubsys(ctx, 12); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
