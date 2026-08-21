package datasources

import (
	"context"
	"testing"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewContainerImagesDataSource(t *testing.T) {
	if NewContainerImagesDataSource() == nil {
		t.Fatal("NewContainerImagesDataSource returned nil")
	}
}

func TestContainerImagesDataSource_Schema(t *testing.T) {
	ds := NewContainerImagesDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "name_prefix", "images", "names", "latest_by_name"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func containerImagesFixture() []truenas.ContainerImage {
	return []truenas.ContainerImage{
		// Deliberately out of order, to prove the sort.
		{Name: "debian:12:amd64:default", Versions: []truenas.ContainerImageVersion{
			{Version: "20260819_00:00"}, {Version: "20260820_00:00"},
		}},
		{Name: "alpine:3.21:amd64:default", Versions: []truenas.ContainerImageVersion{
			{Version: "20260818_13:00"}, {Version: "20260819_13:00"}, {Version: "20260820_13:00"},
		}},
		{Name: "alpine:3.22:amd64:default", Versions: nil},
	}
}

func readContainerImages(t *testing.T, prefix string) ContainerImagesDataSourceModel {
	t.Helper()
	c := newWSServer(t.Context(), t, wsReturn(containerImagesFixture()))
	ds := NewContainerImagesDataSource().(*ContainerImagesDataSource)
	ds.client = c

	vals := map[string]tftypes.Value{}
	if prefix != "" {
		vals["name_prefix"] = strVal(prefix)
	}
	cfg := buildConfig(t.Context(), t, ds, vals)
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state ContainerImagesDataSourceModel
	resp.State.Get(context.Background(), &state)
	return state
}

func TestContainerImagesDataSource_Read(t *testing.T) {
	state := readContainerImages(t, "")

	if state.ID.ValueString() != "container_images" {
		t.Errorf("id = %q", state.ID.ValueString())
	}
	if len(state.Images) != 3 {
		t.Fatalf("got %d images, want 3", len(state.Images))
	}
	// The registry's order is not documented as stable, so the data source
	// sorts; without it the list churns between plans for no reason.
	want := []string{"alpine:3.21:amd64:default", "alpine:3.22:amd64:default", "debian:12:amd64:default"}
	for i, w := range want {
		if got := state.Images[i].Name.ValueString(); got != w {
			t.Errorf("images[%d] = %q, want %q", i, got, w)
		}
	}
	// Versions arrive oldest-first, so the newest is the last one.
	if got := state.Images[0].LatestVersion.ValueString(); got != "20260820_13:00" {
		t.Errorf("latest_version = %q", got)
	}
	if got := len(state.Images[0].Versions.Elements()); got != 3 {
		t.Errorf("versions len = %d", got)
	}
	// An image with no published versions is still listed, with an empty
	// latest_version: dropping it would fail later as a missing map key,
	// which is the harder failure to read.
	if state.Images[1].LatestVersion.IsNull() || state.Images[1].LatestVersion.ValueString() != "" {
		t.Errorf("versionless image latest_version = %v", state.Images[1].LatestVersion)
	}
	if got := len(state.Names.Elements()); got != 3 {
		t.Errorf("names len = %d", got)
	}
	if got := len(state.LatestByName.Elements()); got != 3 {
		t.Errorf("latest_by_name len = %d", got)
	}
}

func TestContainerImagesDataSource_NamePrefixFilters(t *testing.T) {
	state := readContainerImages(t, "alpine:")
	if len(state.Images) != 2 {
		t.Fatalf("got %d images, want the 2 alpine ones", len(state.Images))
	}
	for _, img := range state.Images {
		if got := img.Name.ValueString(); got[:7] != "alpine:" {
			t.Errorf("prefix filter let through %q", got)
		}
	}

	t.Run("a prefix matching nothing is empty, not an error", func(t *testing.T) {
		state := readContainerImages(t, "nosuchimage:")
		if len(state.Images) != 0 {
			t.Errorf("got %d images, want 0", len(state.Images))
		}
	})
}

func TestContainerImagesDataSource_Read_Error(t *testing.T) {
	c := newWSServer(t.Context(), t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: m}
	})
	ds := NewContainerImagesDataSource().(*ContainerImagesDataSource)
	ds.client = c

	resp := callRead(context.Background(), ds, buildConfig(t.Context(), t, ds, nil))
	if !resp.Diagnostics.HasError() {
		t.Fatal("a missing container.image namespace was treated as a successful read")
	}
}

// A config that cannot be decoded must stop the read rather than being
// acted on half-read.
func TestContainerImagesDataSource_undecodableConfigStops(t *testing.T) {
	ctx := context.Background()
	ds := NewContainerImagesDataSource().(*ContainerImagesDataSource)
	ds.client = newWSServer(t.Context(), t, wsReturn(containerImagesFixture()))

	sch := getDataSourceSchema(t.Context(), t, ds)
	bogus := tfsdk.Config{
		Schema: sch.Schema,
		Raw:    tftypes.NewValue(tftypes.String, "not-an-object"),
	}
	resp := callRead(ctx, ds, bogus)
	if !resp.Diagnostics.HasError() {
		t.Error("Read accepted a config it could not decode")
	}
}
