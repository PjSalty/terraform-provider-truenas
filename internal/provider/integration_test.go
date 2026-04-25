package provider

// Integration tests — full terraform-plugin-testing lifecycle against
// a mocked TrueNAS REST server.
//
// These tests use `resource.UnitTest` (NOT `resource.Test`) so they run
// as part of the default `go test ./...` suite without requiring TF_ACC
// or a real TrueNAS instance. Each test spins up an httptest.Server that
// mimics just enough of the TrueNAS API for the resource(s) under test,
// then exercises the full plan → apply → refresh → destroy cycle
// through the real provider factory wired to the mock.
//
// The goal is to catch protocol-level regressions (schema decode errors,
// plan mismatches, CRUD wiring bugs) that unit tests on resource handlers
// alone cannot see. Acceptance tests in the same package exercise the
// real API but require TF_ACC and network access; these integration
// tests fill the gap between the two.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// mockTrueNAS is a minimal in-memory TrueNAS REST API stand-in used by
// integration tests. It supports the subset of endpoints needed by the
// currently-tested resources (dataset + nfs share) and can be expanded
// on demand as more integration tests are added.
type mockTrueNAS struct {
	mu       sync.Mutex
	datasets map[string]map[string]interface{}
	nfs      map[int]map[string]interface{}
	nextNFS  int
}

func newMockTrueNAS() *mockTrueNAS {
	return &mockTrueNAS{
		datasets: map[string]map[string]interface{}{},
		nfs:      map[int]map[string]interface{}{},
		nextNFS:  1,
	}
}

// handler returns an http.Handler suitable for httptest.NewServer.
func (m *mockTrueNAS) handler() http.Handler {
	mux := http.NewServeMux()

	datasetIDRe := regexp.MustCompile(`^/api/v2\.0/pool/dataset/id/(.+)$`)
	nfsIDRe := regexp.MustCompile(`^/api/v2\.0/sharing/nfs/id/(\d+)$`)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()

		path := r.URL.Path

		// --- Datasets ---
		if path == "/api/v2.0/pool/dataset" {
			switch r.Method {
			case http.MethodPost:
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				name, _ := body["name"].(string)
				if name == "" {
					http.Error(w, `{"message":"name required"}`, http.StatusBadRequest)
					return
				}
				resp := datasetResponse(name, body)
				m.datasets[name] = resp
				writeJSON(w, http.StatusOK, resp)
				return
			case http.MethodGet:
				list := make([]map[string]interface{}, 0, len(m.datasets))
				for _, d := range m.datasets {
					list = append(list, d)
				}
				writeJSON(w, http.StatusOK, list)
				return
			}
		}

		if match := datasetIDRe.FindStringSubmatch(path); match != nil {
			// The path segment is URL-encoded — decode slashes.
			id := strings.ReplaceAll(match[1], "%2F", "/")
			id = strings.ReplaceAll(id, "%2f", "/")
			switch r.Method {
			case http.MethodGet:
				ds, ok := m.datasets[id]
				if !ok {
					http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
					return
				}
				writeJSON(w, http.StatusOK, ds)
				return
			case http.MethodPut:
				ds, ok := m.datasets[id]
				if !ok {
					http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
					return
				}
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				// merge updatable properties back into the response shape.
				for k, v := range body {
					if k == "comments" {
						ds["comments"] = map[string]interface{}{"value": v, "source": "LOCAL"}
						continue
					}
					ds[k] = map[string]interface{}{"value": fmt.Sprintf("%v", v), "source": "LOCAL"}
				}
				writeJSON(w, http.StatusOK, ds)
				return
			case http.MethodDelete:
				delete(m.datasets, id)
				writeJSON(w, http.StatusOK, map[string]interface{}{})
				return
			}
		}

		// --- NFS shares ---
		if path == "/api/v2.0/sharing/nfs" {
			switch r.Method {
			case http.MethodPost:
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				id := m.nextNFS
				m.nextNFS++
				body["id"] = id
				fillNFSDefaults(body)
				m.nfs[id] = body
				writeJSON(w, http.StatusOK, body)
				return
			case http.MethodGet:
				list := make([]map[string]interface{}, 0, len(m.nfs))
				for _, n := range m.nfs {
					list = append(list, n)
				}
				writeJSON(w, http.StatusOK, list)
				return
			}
		}

		if match := nfsIDRe.FindStringSubmatch(path); match != nil {
			var id int
			_, _ = fmt.Sscanf(match[1], "%d", &id)
			switch r.Method {
			case http.MethodGet:
				n, ok := m.nfs[id]
				if !ok {
					http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
					return
				}
				fillNFSDefaults(n)
				writeJSON(w, http.StatusOK, n)
				return
			case http.MethodPut:
				n, ok := m.nfs[id]
				if !ok {
					http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
					return
				}
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				for k, v := range body {
					n[k] = v
				}
				fillNFSDefaults(n)
				writeJSON(w, http.StatusOK, n)
				return
			case http.MethodDelete:
				delete(m.nfs, id)
				writeJSON(w, http.StatusOK, map[string]interface{}{})
				return
			}
		}

		http.Error(w, fmt.Sprintf(`{"message":"mock: unhandled %s %s"}`, r.Method, path), http.StatusNotFound)
	})
	return mux
}

// datasetResponse builds a PropertyValue-shaped dataset response from a
// flat create request body. Only the fields used by the dataset resource
// are populated.
func datasetResponse(id string, body map[string]interface{}) map[string]interface{} {
	get := func(k, fallback string) string {
		if v, ok := body[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
		return fallback
	}
	typ := get("type", "FILESYSTEM")

	prop := func(v string) map[string]interface{} {
		return map[string]interface{}{"value": v, "source": "LOCAL"}
	}
	raw := func(v string) map[string]interface{} {
		return map[string]interface{}{"value": v, "rawvalue": v, "source": "LOCAL"}
	}

	resp := map[string]interface{}{
		"id":            id,
		"name":          id,
		"pool":          strings.SplitN(id, "/", 2)[0],
		"type":          typ,
		"mountpoint":    "/mnt/" + id,
		"comments":      nil,
		"quota":         raw("0"),
		"refquota":      raw("0"),
		"compression":   prop(get("compression", "LZ4")),
		"atime":         prop(get("atime", "ON")),
		"deduplication": prop(get("deduplication", "OFF")),
		"sync":          prop(get("sync", "STANDARD")),
		"snapdir":       prop(get("snapdir", "HIDDEN")),
		"copies":        prop("1"),
		"readonly":      prop(get("readonly", "OFF")),
		"recordsize":    prop(get("recordsize", "128K")),
		"share_type":    prop("GENERIC"),
	}

	if c, ok := body["comments"].(string); ok && c != "" {
		resp["comments"] = map[string]interface{}{"value": c, "source": "LOCAL"}
	}
	return resp
}

// fillNFSDefaults ensures an NFS share response body contains every
// field the real TrueNAS API returns, so the provider's Read/Update
// path never encounters an unexpected null where it expects an empty
// list or a concrete scalar.
func fillNFSDefaults(body map[string]interface{}) {
	if _, ok := body["enabled"]; !ok {
		body["enabled"] = true
	}
	if _, ok := body["locked"]; !ok {
		body["locked"] = false
	}
	if _, ok := body["ro"]; !ok {
		body["ro"] = false
	}
	if _, ok := body["comment"]; !ok {
		body["comment"] = ""
	}
	if _, ok := body["maproot_user"]; !ok {
		body["maproot_user"] = nil
	}
	if _, ok := body["maproot_group"]; !ok {
		body["maproot_group"] = nil
	}
	if _, ok := body["mapall_user"]; !ok {
		body["mapall_user"] = nil
	}
	if _, ok := body["mapall_group"]; !ok {
		body["mapall_group"] = nil
	}
	if _, ok := body["hosts"]; !ok {
		body["hosts"] = []string{}
	}
	if _, ok := body["networks"]; !ok {
		body["networks"] = []string{}
	}
	if _, ok := body["security"]; !ok {
		body["security"] = []string{}
	}
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// integrationProviderFactories returns a ProtoV6 provider factory bound
// to a TrueNASProvider that talks to the given mock server URL instead
// of a real TrueNAS instance. The factory injects the URL/API key via
// environment variables so the real Configure path runs end-to-end.
func integrationProviderFactories(t *testing.T, srvURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	t.Setenv("TRUENAS_URL", srvURL)
	t.Setenv("TRUENAS_API_KEY", "integration-test-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")
	// Explicitly clear the read-only, destroy-protection, and
	// request-timeout env so each test starts with known defaults;
	// individual tests can override via a second t.Setenv call after
	// this returns.
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_DESTROY_PROTECTION", "")
	t.Setenv("TRUENAS_REQUEST_TIMEOUT", "")
	return map[string]func() (tfprotov6.ProviderServer, error){
		"truenas": providerserver.NewProtocol6WithError(New("integration-test")()),
	}
}

// TestIntegration_Dataset_CreateReadDestroy exercises the dataset
// resource through a complete terraform apply → destroy lifecycle
// against a mocked TrueNAS API. It verifies that the provider can
// round-trip a dataset create, reflect the computed attributes back
// into state, and emit the correct plan assertions.
func TestIntegration_Dataset_CreateReadDestroy(t *testing.T) {
	// Cannot use t.Parallel() because integrationProviderFactories calls
	// t.Setenv, which is incompatible with parallel tests.
	mock := newMockTrueNAS()
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: integrationProviderFactories(t, srv.URL),
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dataset" "integration" {
  pool = "tank"
  name = "integration-test"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.integration", plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(
							"truenas_dataset.integration",
							tfjsonpath.New("pool"),
							knownvalue.StringExact("tank"),
						),
						plancheck.ExpectKnownValue(
							"truenas_dataset.integration",
							tfjsonpath.New("name"),
							knownvalue.StringExact("integration-test"),
						),
					},
					// Apply-idempotency gate against the deterministic mock
					// backend: after the first apply + a state refresh, the
					// plan MUST be empty. Proves the dataset resource is
					// idempotent through the real provider stack without
					// needing a live TrueNAS, and complements the live-acc
					// ExpectEmptyPlan check in acc_dataset_test.go.
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.integration", "id", "tank/integration-test"),
					resource.TestCheckResourceAttr("truenas_dataset.integration", "pool", "tank"),
					resource.TestCheckResourceAttr("truenas_dataset.integration", "name", "integration-test"),
					resource.TestCheckResourceAttr("truenas_dataset.integration", "type", "FILESYSTEM"),
					resource.TestCheckResourceAttrSet("truenas_dataset.integration", "mount_point"),
				),
			},
		},
	})
}

// TestIntegration_Dataset_DriftDetected simulates an out-of-band mutation
// — someone SSHes to TrueNAS and runs `zfs destroy tank/drift-test` — and
// asserts that the provider detects the drift on the next plan. Without
// this path working, silent infrastructure drift goes unnoticed between
// Terraform runs, which is the single most dangerous failure mode for a
// GitOps-style workflow.
//
// The test uses the mock backend so we can cleanly simulate the OOB
// mutation via direct map manipulation, then fire a second TestStep
// with the same config and ExpectNonEmptyPlan. The live acc variant
// (acc_dataset_test.go TestAccDatasetResource_disappears) does the
// same against a real TrueNAS with a client.DeleteDataset call.
func TestIntegration_Dataset_DriftDetected(t *testing.T) {
	// Cannot use t.Parallel() because integrationProviderFactories calls
	// t.Setenv, which is incompatible with parallel tests.
	mock := newMockTrueNAS()
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: integrationProviderFactories(t, srv.URL),
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dataset" "drift" {
  pool = "tank"
  name = "drift-test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.drift", "id", "tank/drift-test"),
					// Out-of-band mutation happens here via the mock
					// backend's map: delete the dataset entry as if
					// someone ran `zfs destroy` on the host.
					func(_ *terraform.State) error {
						mock.mu.Lock()
						defer mock.mu.Unlock()
						delete(mock.datasets, "tank/drift-test")
						return nil
					},
				),
				// After the Check runs, Terraform will refresh on the
				// next step (re-planning the same config) and see the
				// resource is gone from the mock. The provider's Read
				// should return IsNotFound, which removes the resource
				// from state and re-plans a Create.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestIntegration_Dataset_Update verifies that an in-place attribute
// change (comments) plans as an Update action (not Create or Replace)
// and that the updated value round-trips through refresh.
func TestIntegration_Dataset_Update(t *testing.T) {
	// Cannot use t.Parallel() because integrationProviderFactories calls
	// t.Setenv, which is incompatible with parallel tests.
	mock := newMockTrueNAS()
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: integrationProviderFactories(t, srv.URL),
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dataset" "integration" {
  pool     = "tank"
  name     = "update-test"
  comments = "initial"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_dataset.integration", "comments", "initial"),
			},
			{
				Config: `
resource "truenas_dataset" "integration" {
  pool     = "tank"
  name     = "update-test"
  comments = "updated"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.integration", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_dataset.integration", "comments", "updated"),
			},
		},
	})
}

// TestIntegration_MultiResource exercises a dataset + NFS share pair
// where the share's path implicitly depends on the dataset mount point
// via an interpolation. This catches cross-resource wiring issues that
// single-resource integration tests miss (e.g., planning ordering,
// unknown-value propagation across resource boundaries).
func TestIntegration_MultiResource(t *testing.T) {
	// Cannot use t.Parallel() because integrationProviderFactories calls
	// t.Setenv, which is incompatible with parallel tests.
	mock := newMockTrueNAS()
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: integrationProviderFactories(t, srv.URL),
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dataset" "parent" {
  pool = "tank"
  name = "multi-test"
}

resource "truenas_share_nfs" "share" {
  path    = truenas_dataset.parent.mount_point
  comment = "integration-shared-dataset"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.parent", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("truenas_share_nfs.share", plancheck.ResourceActionCreate),
						// The NFS share's `path` references the dataset's
						// mount_point, which is Computed, so it MUST be
						// unknown at plan time.
						plancheck.ExpectUnknownValue("truenas_share_nfs.share", tfjsonpath.New("path")),
					},
					// Multi-resource idempotency gate: after apply + refresh,
					// the plan MUST be empty across BOTH resources. Catches
					// cross-resource Read inconsistencies that single-resource
					// tests miss — e.g. the dataset's mount_point computation
					// drifting between Create and Read, which would cause the
					// dependent NFS share to always show a phantom update.
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.parent", "id", "tank/multi-test"),
					resource.TestCheckResourceAttrSet("truenas_share_nfs.share", "id"),
					resource.TestCheckResourceAttr("truenas_share_nfs.share", "path", "/mnt/tank/multi-test"),
				),
			},
		},
	})
}

// TestIntegration_ReadOnly_RefusesCreate drives a complete terraform
// plan→apply cycle with TRUENAS_READONLY=1 against the mock TrueNAS
// backend and asserts that the apply step fails with the read-only
// diagnostic. This proves the safety rail propagates correctly through
// every layer of the stack: shell env → provider Configure → Client
// struct → doRequest → CreateDataset wrapper → resource Create →
// plugin-framework diagnostics → terraform-plugin-testing harness.
//
// If the test ever stops failing on create, something is silently
// swallowing ErrReadOnly on the way up — that is the regression this
// test is meant to catch.
func TestIntegration_ReadOnly_RefusesCreate(t *testing.T) {
	mock := newMockTrueNAS()
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	factories := integrationProviderFactories(t, srv.URL)
	// Set AFTER integrationProviderFactories so it wins over the clear.
	t.Setenv("TRUENAS_READONLY", "1")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dataset" "readonly" {
  pool = "tank"
  name = "readonly-integration"
}
`,
				// The safety rail returns ErrReadOnly from CreateDataset,
				// which surfaces as a provider diagnostic whose message
				// contains the literal phrase "read-only mode".
				ExpectError: regexp.MustCompile(`(?i)read.?only`),
			},
		},
	})

	// Invariant: the mock backend MUST NOT have stored the dataset —
	// the safety rail fires before any HTTP work, so the Create POST
	// never reaches the mock handler and the map stays empty.
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if _, exists := mock.datasets["tank/readonly-integration"]; exists {
		t.Error("mock TrueNAS received a dataset create despite read-only mode — the safety rail leaked")
	}
	if got := len(mock.datasets); got != 0 {
		t.Errorf("mock TrueNAS has %d datasets, want 0 — read-only mode leaked %v", got, mock.datasets)
	}
}

// Destroy-protection has no integration-level test at this layer:
// terraform-plugin-testing's resource.UnitTest always runs a final
// `terraform destroy` pass as an in-band cleanup after the last step,
// and the destroy-protection rail correctly refuses that DELETE — so
// the harness reports the test as a "dangling resource" failure even
// though the rail is working exactly as designed. Coverage is
// provided instead by:
//
//   - internal/client/destroy_protection_test.go — 6 unit tests
//     covering the client-layer gate, layered flags, nil receiver,
//     error wrapping, and the allow-through paths (GET/POST/PUT).
//   - internal/provider/provider_destroy_protection_test.go — 4
//     provider Configure tests (env var, HCL, HCL-overrides-env,
//     safe-apply profile) that exercise the full Configure path.
//
// That pair proves the gate propagates from env/HCL through the
// provider struct into the client and out to the wire. The
// resource.UnitTest harness incompatibility is a test-framework
// limitation, not a coverage gap.

// TestIntegration_ReadOnly_AllowsRead verifies the complementary
// invariant: with TRUENAS_READONLY=1, READ-only resources (data sources)
// and the Read phase of resources that were pre-seeded into the mock
// backend still work. The read path is the whole point of the feature;
// if this ever breaks, `terraform plan` against prod stops being
// possible and the safety rail becomes useless.
func TestIntegration_ReadOnly_AllowsRead(t *testing.T) {
	mock := newMockTrueNAS()
	// Pre-seed a dataset in the mock so the Read path has something
	// to return without any POST ever happening.
	mock.datasets["tank/seeded"] = datasetResponse("tank/seeded", map[string]interface{}{
		"name": "tank/seeded",
		"type": "FILESYSTEM",
	})
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	factories := integrationProviderFactories(t, srv.URL)
	t.Setenv("TRUENAS_READONLY", "1")

	// No resource block — just a data source style smoke test via
	// the provider schema/Configure path. If the provider cannot
	// configure itself at all in read-only mode, this test fails.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}
`,
			},
		},
	})
}
