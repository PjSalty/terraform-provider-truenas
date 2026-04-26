package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestGetNetworkConfig_Success(t *testing.T) {
	ctx := context.Background()
	want := client.NetworkConfig{
		ID: 1, Hostname: "truenas", Domain: "example.com",
		Nameserver1: "1.1.1.1", Nameserver2: "8.8.8.8", Nameserver3: "",
		IPv4Gateway: "10.0.0.1", HTTPProxy: "",
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/network/configuration") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetNetworkConfig(ctx)
	if err != nil {
		t.Fatalf("GetNetworkConfig: %v", err)
	}
	if got.Hostname != "truenas" {
		t.Errorf("Hostname: %q", got.Hostname)
	}
	if got.Nameserver1 != "1.1.1.1" {
		t.Errorf("Nameserver1: %q", got.Nameserver1)
	}
}

func TestGetNetworkConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetNetworkConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetNetworkConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.GetNetworkConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing network configuration") {
		t.Errorf("got: %v", err)
	}
}

func TestGetNetworkConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.GetNetworkConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status: %d", apiErr.StatusCode)
	}
}

func TestUpdateNetworkConfig_Success(t *testing.T) {
	ctx := context.Background()
	ns1 := "9.9.9.9"
	ns2 := "8.8.4.4"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.NetworkConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Nameserver1 == nil || *req.Nameserver1 != "9.9.9.9" {
			t.Errorf("Nameserver1 wrong")
		}
		writeJSON(w, http.StatusOK, client.NetworkConfig{ID: 1, Nameserver1: "9.9.9.9", Nameserver2: "8.8.4.4"})
	}))

	got, err := c.UpdateNetworkConfig(ctx, &client.NetworkConfigUpdateRequest{
		Nameserver1: &ns1, Nameserver2: &ns2,
	})
	if err != nil {
		t.Fatalf("UpdateNetworkConfig: %v", err)
	}
	if got.Nameserver1 != "9.9.9.9" {
		t.Errorf("Nameserver1: %q", got.Nameserver1)
	}
}

func TestUpdateNetworkConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad ns"})
	}))
	_, err := c.UpdateNetworkConfig(ctx, &client.NetworkConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "bad ns" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestUpdateNetworkConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "garbage")
	}))
	_, err := c.UpdateNetworkConfig(ctx, &client.NetworkConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing network configuration") {
		t.Errorf("got: %v", err)
	}
}

func TestGetFullNetworkConfig_Success(t *testing.T) {
	ctx := context.Background()
	want := client.FullNetworkConfig{
		ID: 1, Hostname: "host", Domain: "ex.com", IPv4Gateway: "10.0.0.1",
		IPv6Gateway: "::1", Nameserver1: "1.1.1.1", Nameserver2: "8.8.8.8",
		Nameserver3: "", HTTPProxy: "", Hosts: []string{"1.2.3.4 foo", "5.6.7.8 bar"},
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetFullNetworkConfig(ctx)
	if err != nil {
		t.Fatalf("GetFullNetworkConfig: %v", err)
	}
	if got.Hostname != "host" {
		t.Errorf("Hostname: %q", got.Hostname)
	}
	if len(got.Hosts) != 2 {
		t.Errorf("Hosts: %v", got.Hosts)
	}
	if got.IPv6Gateway != "::1" {
		t.Errorf("IPv6Gateway: %q", got.IPv6Gateway)
	}
}

func TestGetFullNetworkConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetFullNetworkConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound")
	}
}

func TestGetFullNetworkConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "bogus")
	}))
	_, err := c.GetFullNetworkConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing full network configuration") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateFullNetworkConfig_Success(t *testing.T) {
	ctx := context.Background()
	hostname := "newhost"
	domain := "new.com"
	gw4 := "198.51.100.1"
	hosts := []string{"1.1.1.1 foo"}

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.FullNetworkConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Hostname == nil || *req.Hostname != "newhost" {
			t.Errorf("Hostname wrong")
		}
		if req.IPv4Gateway == nil || *req.IPv4Gateway != "198.51.100.1" {
			t.Errorf("IPv4Gateway wrong")
		}
		if len(req.Hosts) != 1 {
			t.Errorf("Hosts wrong: %v", req.Hosts)
		}
		writeJSON(w, http.StatusOK, client.FullNetworkConfig{
			ID: 1, Hostname: "newhost", Domain: "new.com",
			IPv4Gateway: "198.51.100.1", Hosts: hosts,
		})
	}))

	got, err := c.UpdateFullNetworkConfig(ctx, &client.FullNetworkConfigUpdateRequest{
		Hostname: &hostname, Domain: &domain, IPv4Gateway: &gw4, Hosts: hosts,
	})
	if err != nil {
		t.Fatalf("UpdateFullNetworkConfig: %v", err)
	}
	if got.Hostname != "newhost" {
		t.Errorf("Hostname: %q", got.Hostname)
	}
}

func TestUpdateFullNetworkConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateFullNetworkConfig(ctx, &client.FullNetworkConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "bad" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestUpdateFullNetworkConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.UpdateFullNetworkConfig(ctx, &client.FullNetworkConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing full network configuration") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateNetworkConfig_OmitEmpty(t *testing.T) {
	ctx := context.Background()
	ns1 := "1.1.1.1"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "nameserver1") {
			t.Errorf("missing ns1: %s", body)
		}
		if strings.Contains(string(body), "nameserver2") {
			t.Errorf("should omit ns2: %s", body)
		}
		writeJSON(w, http.StatusOK, client.NetworkConfig{ID: 1})
	}))
	_, err := c.UpdateNetworkConfig(ctx, &client.NetworkConfigUpdateRequest{Nameserver1: &ns1})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
