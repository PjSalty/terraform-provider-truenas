package datasources

import (
	"context"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestCloudSyncCredentialDataSource_Schema(t *testing.T) {
	ds := NewCloudSyncCredentialDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{"id", "name", "provider_type"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestCloudSyncCredentialDataSource_Read_ByID(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.CloudSyncCredential{
		ID:       5,
		Name:     "my-s3",
		Provider: map[string]interface{}{"type": "S3", "access_key_id": "ABC"},
	}))

	ds := NewCloudSyncCredentialDataSource().(*CloudSyncCredentialDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{
		"id": int64Val(5),
	})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state CloudSyncCredentialDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Name.ValueString() != "my-s3" {
		t.Errorf("Name: got %q", state.Name.ValueString())
	}
	if state.ProviderType.ValueString() != "S3" {
		t.Errorf("ProviderType: got %q", state.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialDataSource_Read_ByName(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn([]truenas.CloudSyncCredential{
		{ID: 1, Name: "first", Provider: map[string]interface{}{"type": "B2"}},
		{ID: 2, Name: "target", Provider: map[string]interface{}{"type": "AZUREBLOB"}},
	}))

	ds := NewCloudSyncCredentialDataSource().(*CloudSyncCredentialDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{
		"name": strVal("target"),
	})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state CloudSyncCredentialDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueInt64() != 2 {
		t.Errorf("ID: got %d", state.ID.ValueInt64())
	}
	if state.ProviderType.ValueString() != "AZUREBLOB" {
		t.Errorf("ProviderType: got %q", state.ProviderType.ValueString())
	}
}

func TestCloudSyncCredentialDataSource_Read_MissingLookupKey(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn([]truenas.CloudSyncCredential{}))

	ds := NewCloudSyncCredentialDataSource().(*CloudSyncCredentialDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, nil)
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing lookup key")
	}
}

func TestCloudSyncCredentialDataSource_Read_NotFoundByID(t *testing.T) {
	c := newWSServer(t.Context(), t, wsError(wsclient.CodeMethodCallError, "[ENOENT] not found"))

	ds := NewCloudSyncCredentialDataSource().(*CloudSyncCredentialDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(999)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for 404")
	}
}

func TestCloudSyncCredentialDataSource_Read_NotFoundByName(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn([]truenas.CloudSyncCredential{{ID: 1, Name: "only"}}))

	ds := NewCloudSyncCredentialDataSource().(*CloudSyncCredentialDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"name": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing credential")
	}
}

func TestCloudSyncCredentialDataSource_Read_EmptyProvider(t *testing.T) {
	// Provider map without "type" key — provider_type should be empty string.
	c := newWSServer(t.Context(), t, wsReturn(truenas.CloudSyncCredential{
		ID:       3,
		Name:     "bare",
		Provider: map[string]interface{}{},
	}))

	ds := NewCloudSyncCredentialDataSource().(*CloudSyncCredentialDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(3)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state CloudSyncCredentialDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ProviderType.ValueString() != "" {
		t.Errorf("ProviderType: got %q, want empty", state.ProviderType.ValueString())
	}
}
