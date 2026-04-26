package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNetworkConfigDataSource_Schema(t *testing.T) {
	ds := NewNetworkConfigDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"hostname", "domain", "nameserver1", "nameserver2", "nameserver3",
		"ipv4gateway", "httpproxy",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestNetworkConfigDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/network/configuration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, client.NetworkConfig{
			ID:          1,
			Hostname:    "truenas",
			Domain:      "local",
			Nameserver1: "1.1.1.1",
			Nameserver2: "8.8.8.8",
			Nameserver3: "",
			IPv4Gateway: "192.168.1.1",
			HTTPProxy:   "",
		})
	}))

	ds := NewNetworkConfigDataSource().(*NetworkConfigDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state NetworkConfigDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Hostname.ValueString() != "truenas" {
		t.Errorf("Hostname: got %q", state.Hostname.ValueString())
	}
	if state.Nameserver1.ValueString() != "1.1.1.1" {
		t.Errorf("Nameserver1: got %q", state.Nameserver1.ValueString())
	}
	if state.IPv4Gateway.ValueString() != "192.168.1.1" {
		t.Errorf("IPv4Gateway: got %q", state.IPv4Gateway.ValueString())
	}
}

func TestNetworkConfigDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewNetworkConfigDataSource().(*NetworkConfigDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestNetworkConfigDataSource_Read_InvalidJSON(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))

	ds := NewNetworkConfigDataSource().(*NetworkConfigDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}
