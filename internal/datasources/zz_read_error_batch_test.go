package datasources

// Read-error batch: every id-keyed datasource's Read must surface a
// diagnostic when the upstream call fails. Plus the nil-vs-populated
// settings/attributes JSON branches for the JSON-carrying datasources.

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func wsFailAll(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSServer(ctx, t, wsError(wsclient.CodeMethodCallError, "simulated failure"))
}

func TestDataSourceReadError_Batch(t *testing.T) {
	ctx := context.Background()

	t.Run("alert_service", func(t *testing.T) {
		ds := NewAlertServiceDataSource().(*AlertServiceDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(3)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("api_key", func(t *testing.T) {
		ds := NewAPIKeyDataSource().(*APIKeyDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(2)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("iscsi_extent", func(t *testing.T) {
		ds := NewISCSIExtentDataSource().(*ISCSIExtentDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(15)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("iscsi_initiator", func(t *testing.T) {
		ds := NewISCSIInitiatorDataSource().(*ISCSIInitiatorDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(16)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("iscsi_portal", func(t *testing.T) {
		ds := NewISCSIPortalDataSource().(*ISCSIPortalDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(17)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("iscsi_target", func(t *testing.T) {
		ds := NewISCSITargetDataSource().(*ISCSITargetDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(18)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("keychain_credential", func(t *testing.T) {
		ds := NewKeychainCredentialDataSource().(*KeychainCredentialDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(22)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("pool", func(t *testing.T) {
		ds := NewPoolDataSource().(*PoolDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"name": strVal("gone")}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})

	t.Run("snapshot_task", func(t *testing.T) {
		ds := NewSnapshotTaskDataSource().(*SnapshotTaskDataSource)
		ds.client = wsFailAll(ctx, t)
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(36)}))
		if !resp.Diagnostics.HasError() {
			t.Error("expected diagnostics")
		}
	})
}

// Nil-vs-populated JSON branches.
func TestDataSourceNilJSON_Branches(t *testing.T) {
	ctx := context.Background()

	t.Run("alert_service nil settings", func(t *testing.T) {
		ds := NewAlertServiceDataSource().(*AlertServiceDataSource)
		ds.client = newWSServer(ctx, t, wsReturn(map[string]interface{}{
			"id": 3, "name": "m", "type": "Mail", "enabled": true,
			"level": "INFO", "attributes": nil,
		}))
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(3)}))
		if resp.Diagnostics.HasError() {
			t.Errorf("Read: %v", resp.Diagnostics)
		}
	})

	t.Run("keychain nil attributes", func(t *testing.T) {
		ds := NewKeychainCredentialDataSource().(*KeychainCredentialDataSource)
		ds.client = newWSServer(ctx, t, wsReturn(map[string]interface{}{
			"id": 22, "name": "k", "type": "SSH_KEY_PAIR", "attributes": nil,
		}))
		resp := callRead(ctx, ds, buildConfig(ctx, t, ds, map[string]tftypes.Value{"id": int64Val(22)}))
		if resp.Diagnostics.HasError() {
			t.Errorf("Read: %v", resp.Diagnostics)
		}
	})
}
