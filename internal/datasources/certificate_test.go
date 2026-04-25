package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestCertificateDataSource_Schema(t *testing.T) {
	ds := NewCertificateDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "name", "issuer", "valid_from", "valid_until", "san",
		"key_type", "key_length", "digest_algorithm", "expired", "dn",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestCertificateDataSource_Read_Success(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		certs := []client.Certificate{
			{
				ID:              7,
				Name:            "wildcard",
				Common:          "*.example.com",
				From:            "2026-01-01",
				Until:           "2027-01-01",
				KeyType:         "RSA",
				KeyLength:       2048,
				DigestAlgorithm: "SHA256",
				Expired:         false,
				DN:              "CN=*.example.com",
				SAN:             []string{"*.example.com", "example.com"},
			},
			{ID: 8, Name: "other"},
		}
		writeJSON(w, http.StatusOK, certs)
	}))

	ds := NewCertificateDataSource().(*CertificateDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"name": strVal("wildcard"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state CertificateDataSourceModel
	d := resp.State.Get(context.Background(), &state)
	if d.HasError() {
		t.Fatalf("State.Get: %v", d)
	}
	if state.ID.ValueInt64() != 7 {
		t.Errorf("ID: got %d", state.ID.ValueInt64())
	}
	if state.Issuer.ValueString() != "*.example.com" {
		t.Errorf("Issuer: got %q", state.Issuer.ValueString())
	}
	if state.SAN.ValueString() != "*.example.com,example.com" {
		t.Errorf("SAN: got %q", state.SAN.ValueString())
	}
	if state.KeyLength.ValueInt64() != 2048 {
		t.Errorf("KeyLength: got %d", state.KeyLength.ValueInt64())
	}
	if state.Expired.ValueBool() != false {
		t.Errorf("Expired: got %v", state.Expired.ValueBool())
	}
}

func TestCertificateDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Certificate{{ID: 1, Name: "other"}})
	}))

	ds := NewCertificateDataSource().(*CertificateDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"name": strVal("missing"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing cert")
	}
}

func TestCertificateDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewCertificateDataSource().(*CertificateDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{
		"name": strVal("any"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error on server error")
	}
}

func TestCertificateDataSource_Read_EmptySAN(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.Certificate{
			{ID: 1, Name: "bare", Common: "x", SAN: nil},
		})
	}))

	ds := NewCertificateDataSource().(*CertificateDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"name": strVal("bare")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state CertificateDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.SAN.ValueString() != "" {
		t.Errorf("SAN: got %q, want empty", state.SAN.ValueString())
	}
}
