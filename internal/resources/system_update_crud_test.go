package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// systemUpdateHandler returns an http.HandlerFunc that multiplexes every
// TrueNAS update-related endpoint exercised by the resource. The `fail`
// map lets a test force a specific path to return 500 so the error branches
// in refreshState and applyConfig are covered deterministically.
func systemUpdateHandler(t *testing.T, trains *struct {
	Trains   map[string]map[string]string `json:"trains"`
	Current  string                       `json:"current"`
	Selected string                       `json:"selected"`
}, autoDownload *bool, sysVersion string, checkStatus string, checkVersion string, fail map[string]bool) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if fail[r.URL.Path] {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"message":"forced failure"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/update/get_auto_download"):
			_ = json.NewEncoder(w).Encode(*autoDownload)
		case strings.HasSuffix(r.URL.Path, "/update/get_trains"):
			_ = json.NewEncoder(w).Encode(trains)
		case strings.HasSuffix(r.URL.Path, "/system/info"):
			_ = json.NewEncoder(w).Encode(map[string]string{"version": sysVersion})
		case strings.HasSuffix(r.URL.Path, "/update/check_available"):
			body := map[string]string{"status": checkStatus}
			if checkVersion != "" {
				body["version"] = checkVersion
			}
			_ = json.NewEncoder(w).Encode(body)
		case strings.HasSuffix(r.URL.Path, "/update/set_auto_download"):
			if r.Method != http.MethodPost {
				t.Errorf("set_auto_download wrong method: %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/update/set_train"):
			if r.Method != http.MethodPost {
				t.Errorf("set_train wrong method: %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// stdTrains returns a default trains response mirroring the prod topology.
func stdTrains() *struct {
	Trains   map[string]map[string]string `json:"trains"`
	Current  string                       `json:"current"`
	Selected string                       `json:"selected"`
} {
	return &struct {
		Trains   map[string]map[string]string `json:"trains"`
		Current  string                       `json:"current"`
		Selected string                       `json:"selected"`
	}{
		Trains: map[string]map[string]string{
			"TrueNAS-SCALE-Fangtooth": {"description": "Fangtooth 25.04 [release]"},
			"TrueNAS-SCALE-Goldeye":   {"description": "Goldeye 25.10"},
		},
		Current:  "TrueNAS-SCALE-Fangtooth",
		Selected: "TrueNAS-SCALE-Fangtooth",
	}
}

// primedPlanWithSystemUpdate builds a tfsdk.Plan with auto_download and
// train attributes set to the given values, all other (computed) attributes
// left as null. Used to drive Create and Update flows.
func primedPlanWithSystemUpdate(t *testing.T, ctx context.Context, r *SystemUpdateResource, autoDownload bool, train string) resource.CreateRequest {
	t.Helper()
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	typ := sch.Schema.Type().TerraformType(ctx)
	objType := typ.(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		switch name {
		case "auto_download":
			vals[name] = tftypes.NewValue(at, autoDownload)
		case "train":
			if train == "" {
				vals[name] = tftypes.NewValue(at, nil)
			} else {
				vals[name] = tftypes.NewValue(at, train)
			}
		default:
			vals[name] = tftypes.NewValue(at, nil)
		}
	}
	return resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: sch.Schema, Raw: tftypes.NewValue(objType, vals)},
	}
}

func TestSystemUpdateResource_Metadata(t *testing.T) {
	r := &SystemUpdateResource{}
	md := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "truenas"}, md)
	if md.TypeName != "truenas_system_update" {
		t.Errorf("TypeName: %q", md.TypeName)
	}
}

func TestSystemUpdateResource_Schema(t *testing.T) {
	r := &SystemUpdateResource{}
	sch := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, sch)
	want := []string{"id", "auto_download", "train", "current_version", "available_status", "available_version"}
	for _, attr := range want {
		if _, ok := sch.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute: %q", attr)
		}
	}
}

func TestSystemUpdateResource_Read_Success(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", nil))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedStateWithID(t, ctx, *sch, systemUpdateSingletonID)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}

	var got SystemUpdateResourceModel
	readResp.State.Get(ctx, &got)
	if got.CurrentVersion.ValueString() != "25.04.2.6" {
		t.Errorf("CurrentVersion: %q", got.CurrentVersion.ValueString())
	}
	if got.Train.ValueString() != "TrueNAS-SCALE-Fangtooth" {
		t.Errorf("Train: %q", got.Train.ValueString())
	}
	if got.AvailableStatus.ValueString() != "UNAVAILABLE" {
		t.Errorf("AvailableStatus: %q", got.AvailableStatus.ValueString())
	}
	if got.AutoDownload.ValueBool() {
		t.Errorf("AutoDownload should be false")
	}
}

func TestSystemUpdateResource_Read_ErrorsEachBranch(t *testing.T) {
	ctx := context.Background()
	ad := true
	cases := []struct {
		name string
		fail map[string]bool
	}{
		{"get_auto_download fails", map[string]bool{"/api/v2.0/update/get_auto_download": true}},
		{"get_trains fails", map[string]bool{"/api/v2.0/update/get_trains": true}},
		{"system/info fails", map[string]bool{"/api/v2.0/system/info": true}},
		{"check_available fails", map[string]bool{"/api/v2.0/update/check_available": true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "AVAILABLE", "25.04.3.0", tc.fail))
			defer srv.Close()
			r := &SystemUpdateResource{client: c}
			sch := &resource.SchemaResponse{}
			r.Schema(ctx, resource.SchemaRequest{}, sch)
			state := primedStateWithID(t, ctx, *sch, systemUpdateSingletonID)
			readResp := &resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, readResp)
			if !readResp.Diagnostics.HasError() {
				t.Errorf("expected error diagnostic for %s", tc.name)
			}
		})
	}
}

func TestSystemUpdateResource_Create_Success_PinsNewTrain(t *testing.T) {
	ctx := context.Background()
	ad := false
	trains := stdTrains()
	trains.Selected = "TrueNAS-SCALE-Fangtooth"
	c, srv := newTestServerClient(t, systemUpdateHandler(t, trains, &ad, "25.04.2.6", "UNAVAILABLE", "", nil))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	req := primedPlanWithSystemUpdate(t, ctx, r, false, "TrueNAS-SCALE-Goldeye")

	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
}

func TestSystemUpdateResource_Create_Success_TrainAlreadySelected(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", nil))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	// Train matches trains.Selected → SetUpdateTrain should be skipped.
	req := primedPlanWithSystemUpdate(t, ctx, r, true, "TrueNAS-SCALE-Fangtooth")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
}

func TestSystemUpdateResource_Create_Success_NoTrain(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", nil))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	// No train supplied in plan → applyConfig skips the train block entirely.
	req := primedPlanWithSystemUpdate(t, ctx, r, false, "")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
}

func TestSystemUpdateResource_Create_InvalidTrain(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", nil))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	req := primedPlanWithSystemUpdate(t, ctx, r, false, "TrueNAS-SCALE-Nonexistent")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected error for invalid train")
	}
}

func TestSystemUpdateResource_Create_GetTrainsFails(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", map[string]bool{"/api/v2.0/update/get_trains": true}))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	req := primedPlanWithSystemUpdate(t, ctx, r, false, "TrueNAS-SCALE-Fangtooth")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected error when get_trains fails")
	}
}

func TestSystemUpdateResource_Create_SetTrainFails(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", map[string]bool{"/api/v2.0/update/set_train": true}))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	// Requesting a train DIFFERENT from trains.Selected forces SetUpdateTrain.
	req := primedPlanWithSystemUpdate(t, ctx, r, false, "TrueNAS-SCALE-Goldeye")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected error when set_train fails")
	}
}

func TestSystemUpdateResource_Create_SetAutoDownloadFails(t *testing.T) {
	ctx := context.Background()
	ad := false
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", map[string]bool{"/api/v2.0/update/set_auto_download": true}))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	req := primedPlanWithSystemUpdate(t, ctx, r, true, "")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected error when set_auto_download fails")
	}
}

func TestSystemUpdateResource_Create_RefreshStateFails(t *testing.T) {
	ctx := context.Background()
	ad := false
	// applyConfig succeeds, but the subsequent refreshState hits a failing
	// get_auto_download. This exercises the "writes OK, reads fail" path.
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", map[string]bool{"/api/v2.0/update/get_auto_download": true}))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	req := primedPlanWithSystemUpdate(t, ctx, r, false, "")
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.CreateResponse{State: state}
	r.Create(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected error when refresh fails after apply")
	}
}

func TestSystemUpdateResource_Update_Success(t *testing.T) {
	ctx := context.Background()
	ad := true
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", nil))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedStateWithID(t, ctx, *sch, systemUpdateSingletonID)
	plan := primedPlanWithSystemUpdate(t, ctx, r, false, "").Plan
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
}

func TestSystemUpdateResource_Update_ApplyFails(t *testing.T) {
	ctx := context.Background()
	ad := true
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", map[string]bool{"/api/v2.0/update/set_auto_download": true}))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedStateWithID(t, ctx, *sch, systemUpdateSingletonID)
	plan := primedPlanWithSystemUpdate(t, ctx, r, false, "").Plan
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Fatalf("expected error when apply fails in Update")
	}
}

func TestSystemUpdateResource_Update_RefreshFails(t *testing.T) {
	ctx := context.Background()
	ad := true
	c, srv := newTestServerClient(t, systemUpdateHandler(t, stdTrains(), &ad, "25.04.2.6", "UNAVAILABLE", "", map[string]bool{"/api/v2.0/system/info": true}))
	defer srv.Close()

	r := &SystemUpdateResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedStateWithID(t, ctx, *sch, systemUpdateSingletonID)
	plan := primedPlanWithSystemUpdate(t, ctx, r, false, "").Plan
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Fatalf("expected error when refresh fails in Update")
	}
}

func TestSystemUpdateResource_Delete_NoOp(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}
	// Delete is a no-op — no client calls, no state mutation. Just prove
	// it doesn't produce a diagnostic or panic.
	resp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", resp.Diagnostics)
	}
}

func TestSystemUpdateResource_ImportState_Valid(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	resp := &resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: systemUpdateSingletonID}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("ImportState(valid): %v", resp.Diagnostics)
	}
}

func TestSystemUpdateResource_ImportState_Invalid(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)
	state := primedState(t, ctx, *sch)
	resp := &resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "not-the-singleton"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Errorf("ImportState(invalid) should have errored")
	}
}
