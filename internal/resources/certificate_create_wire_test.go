package resources

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// captureCertCreate serves certificate.create as a job and records the params
// it was given, so a test can assert what actually went on the wire.
func captureCertCreate(ctx context.Context, t *testing.T, body map[string]interface{}) (*wsclient.Client, func() map[string]interface{}) {
	t.Helper()
	var mu sync.Mutex
	var seen map[string]interface{}

	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "system.info":
			return map[string]interface{}{"version": "26.0.0-BETA.2"}, nil
		case "certificate.create":
			mu.Lock()
			if len(params) > 0 {
				// The client marshals the typed request struct, so round-trip
				// through JSON to see exactly the keys that were sent.
				b, err := json.Marshal(params[0])
				if err == nil {
					_ = json.Unmarshal(b, &seen)
				}
			}
			mu.Unlock()
			return 1, nil // job id
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": 1, "state": "SUCCESS", "result": body, "error": "",
			}}, nil
		}
		return body, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c, func() map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		return seen
	}
}

// The five ACME fields have to reach the wire. Their absence is what made
// create_type=CERTIFICATE_CREATE_ACME impossible: the request was well-formed
// as far as the provider was concerned and the server rejected three fields
// nobody could set.
func TestCertificateResource_CreateSendsACMEFields(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertCreate(ctx, t, map[string]interface{}{
		"id": 9, "name": "acme", "key_type": "RSA", "key_length": 4096,
		"digest_algorithm": "SHA256", "san": []interface{}{"DNS:example.com"},
	})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":               str("acme"),
		"create_type":        str("CERTIFICATE_CREATE_ACME"),
		"tos":                tftypes.NewValue(tftypes.Bool, true),
		"csr_id":             num(4),
		"acme_directory_uri": str("https://acme-v02.api.letsencrypt.org/directory"),
		"renew_days":         num(15),
		"dns_mapping": tftypes.NewValue(
			tftypes.Map{ElementType: tftypes.Number},
			map[string]tftypes.Value{"example.com": num(3)},
		),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	got := sent()
	if got == nil {
		t.Fatal("certificate.create was never called")
	}
	for k, want := range map[string]interface{}{
		"tos":                true,
		"csr_id":             float64(4),
		"acme_directory_uri": "https://acme-v02.api.letsencrypt.org/directory",
		"renew_days":         float64(15),
	} {
		if got[k] != want {
			t.Errorf("%s on the wire = %#v, want %#v", k, got[k], want)
		}
	}
	m, ok := got["dns_mapping"].(map[string]interface{})
	if !ok {
		t.Fatalf("dns_mapping on the wire = %#v, want a map", got["dns_mapping"])
	}
	if m["example.com"] != float64(3) {
		t.Errorf("dns_mapping[example.com] = %#v, want 3", m["example.com"])
	}
}

// tos = false has to survive. It is a *bool precisely so that `omitempty` does
// not drop it: a false that never arrives reads to the server as absent, which
// is the "Input should be a valid boolean" the issue reported.
func TestCertificateResource_CreateSendsFalseTOS(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertCreate(ctx, t, map[string]interface{}{
		"id": 10, "name": "acme-no-tos", "key_type": "RSA", "key_length": 4096,
	})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":               str("acme-no-tos"),
		"create_type":        str("CERTIFICATE_CREATE_ACME"),
		"tos":                tftypes.NewValue(tftypes.Bool, false),
		"csr_id":             num(4),
		"acme_directory_uri": str("https://acme-staging-v02.api.letsencrypt.org/directory"),
		"dns_mapping": tftypes.NewValue(
			tftypes.Map{ElementType: tftypes.Number},
			map[string]tftypes.Value{"example.com": num(3)},
		),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	got := sent()
	if _, present := got["tos"]; !present {
		t.Fatal("tos = false was dropped from the request")
	}
	if got["tos"] != false {
		t.Errorf("tos on the wire = %#v, want false", got["tos"])
	}
	// renew_days was not set, so it must not be sent at all: the server
	// defaults it to 10 and an explicit zero would be out of range.
	if _, present := got["renew_days"]; present {
		t.Errorf("renew_days was sent as %#v despite not being configured", got["renew_days"])
	}
}

// A CSR sends its SANs bare. TrueNAS renders them back with a general-name
// kind attached, and sending that back is not what was asked for.
func TestCertificateResource_CreateStripsSANPrefixes(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertCreate(ctx, t, map[string]interface{}{
		"id": 11, "name": "csr", "key_type": "RSA", "key_length": 4096,
		"digest_algorithm": "SHA256",
		"san":              []interface{}{"DNS:example.com", "DNS:www.example.com"},
	})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":             str("csr"),
		"create_type":      str("CERTIFICATE_CREATE_CSR"),
		"key_type":         str("RSA"),
		"key_length":       num(4096),
		"digest_algorithm": str("SHA256"),
		"san": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.String},
			[]tftypes.Value{str("DNS:example.com"), str("www.example.com")},
		),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	got := sent()
	san, ok := got["san"].([]interface{})
	if !ok {
		t.Fatalf("san on the wire = %#v, want a list", got["san"])
	}
	want := []string{"example.com", "www.example.com"}
	if len(san) != len(want) {
		t.Fatalf("san on the wire = %#v, want %v", san, want)
	}
	for i, w := range want {
		if san[i] != w {
			t.Errorf("san[%d] on the wire = %#v, want %q", i, san[i], w)
		}
	}
}

// captureCertUpdate is captureCertCreate for certificate.update, which is a
// plain call rather than a job.
func captureCertUpdate(ctx context.Context, t *testing.T, body map[string]interface{}) (*wsclient.Client, func() map[string]interface{}) {
	t.Helper()
	var mu sync.Mutex
	var seen map[string]interface{}

	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "system.info":
			return map[string]interface{}{"version": "26.0.0-BETA.2"}, nil
		case "certificate.update":
			mu.Lock()
			if len(params) > 1 {
				if b, err := json.Marshal(params[1]); err == nil {
					_ = json.Unmarshal(b, &seen)
				}
			}
			mu.Unlock()
			return 2, nil // job id; certificate.update is a job too
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": 2, "state": "SUCCESS", "result": body, "error": "",
			}}, nil
		}
		return body, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c, func() map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		return seen
	}
}

// renew_days is the one ACME field upstream's update model accepts, so it has
// to actually be sent. Before this it was not, and Update writes the plan into
// state, so changing it left state claiming a value the server never got.
func TestCertificateResource_UpdateSendsRenewDays(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertUpdate(ctx, t, map[string]interface{}{
		"id": 12, "name": "acme", "key_type": "RSA", "key_length": 4096,
	})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	vals := map[string]tftypes.Value{
		"id":                 str("12"),
		"name":               str("acme"),
		"create_type":        str("CERTIFICATE_CREATE_ACME"),
		"tos":                tftypes.NewValue(tftypes.Bool, true),
		"csr_id":             num(4),
		"acme_directory_uri": str("https://acme-v02.api.letsencrypt.org/directory"),
		"renew_days":         num(20),
		"dns_mapping": tftypes.NewValue(
			tftypes.Map{ElementType: tftypes.Number},
			map[string]tftypes.Value{"example.com": num(3)},
		),
	}
	state := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}

	got := sent()
	if got == nil {
		t.Fatal("certificate.update was never called")
	}
	if got["renew_days"] != float64(20) {
		t.Errorf("renew_days on the wire = %#v, want 20", got["renew_days"])
	}
	if got["name"] != "acme" {
		t.Errorf("name on the wire = %#v", got["name"])
	}
}

// An unset renew_days must not be sent: 0 is outside the 1..30 the server
// accepts, so a zero value would turn a rename into a validation error.
func TestCertificateResource_UpdateOmitsUnsetRenewDays(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertUpdate(ctx, t, map[string]interface{}{
		"id": 13, "name": "imported-v2", "key_type": "RSA", "key_length": 4096,
	})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	vals := map[string]tftypes.Value{
		"id":          str("13"),
		"name":        str("imported-v2"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate": str("-----BEGIN CERTIFICATE-----"),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
	}
	state := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}

	got := sent()
	if _, present := got["renew_days"]; present {
		t.Errorf("renew_days was sent as %#v despite not being configured", got["renew_days"])
	}
}

// CERTIFICATE_CREATE_IMPORTED_CSR sends the request itself, under the key the
// server uses. Its payload model is name + CSR + privatekey, and the provider
// carried no csr attribute at all, so the create_type was advertised by the
// schema and could never succeed.
func TestCertificateResource_CreateSendsImportedCSR(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertCreate(ctx, t, map[string]interface{}{
		"id": 14, "name": "imported-csr", "key_type": "RSA", "key_length": 4096,
		"CSR": "-----BEGIN CERTIFICATE REQUEST-----",
	})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":        str("imported-csr"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED_CSR"),
		"csr":         str("-----BEGIN CERTIFICATE REQUEST-----"),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	got := sent()
	// Upstream spells it CSR, not csr; a lowercase key is an unknown field to
	// a model that forbids extras.
	if got["CSR"] != "-----BEGIN CERTIFICATE REQUEST-----" {
		t.Errorf("CSR on the wire = %#v", got["CSR"])
	}
	if _, present := got["csr"]; present {
		t.Error("the request carried a lowercase csr key")
	}
}

// ImportState used to assume CERTIFICATE_CREATE_IMPORTED for everything, and
// create_type forces replacement, so importing a CSR or an ACME certificate
// and planning against the configuration that describes it proposed a destroy
// and re-create. For an ACME certificate that means re-issuing from the CA.
func TestCertificateResource_ImportInfersCreateType(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		body map[string]interface{}
		want string
	}{
		{
			name: "a request that has not been signed",
			body: map[string]interface{}{"id": 1, "name": "csr", "cert_type_CSR": true},
			want: "CERTIFICATE_CREATE_CSR",
		},
		{
			name: "issued through ACME",
			body: map[string]interface{}{
				"id": 2, "name": "acme",
				"acme_uri": "https://acme-v02.api.letsencrypt.org/directory",
			},
			want: "CERTIFICATE_CREATE_ACME",
		},
		{
			name: "neither, so imported",
			body: map[string]interface{}{"id": 3, "name": "imported"},
			want: "CERTIFICATE_CREATE_IMPORTED",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := captureCertCreate(ctx, t, tc.body)
			r := &CertificateResource{client: c}
			sch := schemaOf(t, ctx, r)

			resp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, sch)}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ImportState: %v", resp.Diagnostics)
			}
			var got types.String
			resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("create_type"), &got)...)
			if resp.Diagnostics.HasError() {
				t.Fatalf("reading create_type: %v", resp.Diagnostics)
			}
			if got.ValueString() != tc.want {
				t.Errorf("create_type = %q, want %q", got.ValueString(), tc.want)
			}
		})
	}
}

// A conversion failure must stop the create. It used to only append a
// diagnostic, so the request went out with the failed field missing and the
// server made a certificate Terraform then had no state for.
func TestCertificateResource_CreateStopsOnConversionError(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertCreate(ctx, t, map[string]interface{}{"id": 20, "name": "x"})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	// A san list carrying a null element cannot convert into []string.
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":             str("x"),
		"create_type":      str("CERTIFICATE_CREATE_CSR"),
		"key_type":         str("RSA"),
		"key_length":       num(4096),
		"digest_algorithm": str("SHA256"),
		"san": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.String},
			[]tftypes.Value{tftypes.NewValue(tftypes.String, nil)},
		),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a null san element converted without complaint")
	}
	if sent() != nil {
		t.Error("certificate.create was issued despite the conversion error")
	}
}

// A read failure during import must not fail the import: create_type is a
// hint, and the Read that follows reports the real problem.
func TestCertificateResource_ImportToleratesReadFailure(t *testing.T) {
	ctx := context.Background()
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, _ []interface{}) (interface{}, *wsclient.RPCError) {
		if method == "system.info" {
			return map[string]interface{}{"version": "26.0.0-BETA.2"}, nil
		}
		return nil, &wsclient.RPCError{Code: -32000, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	resp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, sch)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a read failure broke the import: %v", resp.Diagnostics)
	}
	var got types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("create_type"), &got)...)
	if got.ValueString() != "CERTIFICATE_CREATE_IMPORTED" {
		t.Errorf("create_type = %q, want the imported fallback", got.ValueString())
	}
}

// A non-numeric id is refused before any call goes out.
func TestCertificateResource_ImportRejectsNonNumericID(t *testing.T) {
	ctx := context.Background()
	c, sent := captureCertCreate(ctx, t, map[string]interface{}{"id": 1})
	r := &CertificateResource{client: c}
	sch := schemaOf(t, ctx, r)

	resp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, sch)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "not-a-number"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a non-numeric import id was accepted")
	}
	if sent() != nil {
		t.Error("a call went out for an invalid id")
	}
}
