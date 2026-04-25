package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// dataSourceCase describes a data source under test.
type dataSourceCase struct {
	name        string
	newFn       func() datasource.DataSource
	wantTypeSfx string // suffix after provider prefix
}

// allDataSources returns a table of every data source and its expected
// TypeName suffix. This is used to exercise Metadata and Configure for every
// data source from one place, which avoids duplicating boilerplate in 25
// separate files.
func allDataSources() []dataSourceCase {
	return []dataSourceCase{
		{"app", NewAppDataSource, "_app"},
		{"apps", NewAppsDataSource, "_apps"},
		{"catalog", NewCatalogDataSource, "_catalog"},
		{"certificate", NewCertificateDataSource, "_certificate"},
		{"cloudsync_credential", NewCloudSyncCredentialDataSource, "_cloudsync_credential"},
		{"cronjob", NewCronJobDataSource, "_cronjob"},
		{"dataset", NewDatasetDataSource, "_dataset"},
		{"datasets", NewDatasetsDataSource, "_datasets"},
		{"directoryservices", NewDirectoryServicesDataSource, "_directoryservices"},
		{"disk", NewDiskDataSource, "_disk"},
		{"group", NewGroupDataSource, "_group"},
		{"kerberos_realm", NewKerberosRealmDataSource, "_kerberos_realm"},
		{"network_config", NewNetworkConfigDataSource, "_network_config"},
		{"network_interface", NewNetworkInterfaceDataSource, "_network_interface"},
		{"pool", NewPoolDataSource, "_pool"},
		{"pools", NewPoolsDataSource, "_pools"},
		{"privilege", NewPrivilegeDataSource, "_privilege"},
		{"service", NewServiceDataSource, "_service"},
		{"share_nfs", NewShareNFSDataSource, "_share_nfs"},
		{"share_smb", NewShareSMBDataSource, "_share_smb"},
		{"system_info", NewSystemInfoDataSource, "_system_info"},
		{"systemdataset", NewSystemDatasetDataSource, "_systemdataset"},
		{"user", NewUserDataSource, "_user"},
		{"vm", NewVMDataSource, "_vm"},
		{"vms", NewVMsDataSource, "_vms"},
		// v0.5.0 — parity expansion
		{"iscsi_target", NewISCSITargetDataSource, "_iscsi_target"},
		{"iscsi_portal", NewISCSIPortalDataSource, "_iscsi_portal"},
		{"iscsi_extent", NewISCSIExtentDataSource, "_iscsi_extent"},
		{"iscsi_initiator", NewISCSIInitiatorDataSource, "_iscsi_initiator"},
		{"api_key", NewAPIKeyDataSource, "_api_key"},
		{"keychain_credential", NewKeychainCredentialDataSource, "_keychain_credential"},
		{"snapshot_task", NewSnapshotTaskDataSource, "_snapshot_task"},
		{"alert_service", NewAlertServiceDataSource, "_alert_service"},
	}
}

// TestAllDataSources_Metadata verifies that every data source reports the
// expected TypeName given the provider prefix.
func TestAllDataSources_Metadata(t *testing.T) {
	const prefix = "truenas"
	for _, tc := range allDataSources() {
		t.Run(tc.name, func(t *testing.T) {
			ds := tc.newFn()
			req := datasource.MetadataRequest{ProviderTypeName: prefix}
			resp := datasource.MetadataResponse{}
			ds.Metadata(context.Background(), req, &resp)
			want := prefix + tc.wantTypeSfx
			if resp.TypeName != want {
				t.Errorf("TypeName: got %q, want %q", resp.TypeName, want)
			}
		})
	}
}

// TestAllDataSources_Configure_Nil ensures Configure tolerates a nil
// ProviderData (the framework invokes Configure before the provider has been
// fully configured, with ProviderData == nil, and data sources must no-op).
func TestAllDataSources_Configure_Nil(t *testing.T) {
	for _, tc := range allDataSources() {
		t.Run(tc.name, func(t *testing.T) {
			ds := tc.newFn()
			cfg, ok := ds.(datasource.DataSourceWithConfigure)
			if !ok {
				t.Fatalf("data source %s does not implement DataSourceWithConfigure", tc.name)
			}
			req := datasource.ConfigureRequest{ProviderData: nil}
			resp := datasource.ConfigureResponse{}
			cfg.Configure(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected diagnostics for nil ProviderData: %v", resp.Diagnostics)
			}
		})
	}
}

// TestAllDataSources_Configure_WrongType asserts that passing a ProviderData
// that is not a *client.Client produces an error diagnostic.
func TestAllDataSources_Configure_WrongType(t *testing.T) {
	for _, tc := range allDataSources() {
		t.Run(tc.name, func(t *testing.T) {
			ds := tc.newFn()
			cfg, ok := ds.(datasource.DataSourceWithConfigure)
			if !ok {
				t.Fatalf("data source %s does not implement DataSourceWithConfigure", tc.name)
			}
			// int is unambiguously not *client.Client.
			req := datasource.ConfigureRequest{ProviderData: 42}
			resp := datasource.ConfigureResponse{}
			cfg.Configure(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error diagnostic for wrong ProviderData type")
			}
		})
	}
}

// TestAllDataSources_Configure_Client asserts that a valid *client.Client is
// stored without error. We cannot inspect the private client field in a
// generic way, but exercising the happy path still counts toward coverage.
func TestAllDataSources_Configure_Client(t *testing.T) {
	c, err := client.New("http://example.invalid", "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	for _, tc := range allDataSources() {
		t.Run(tc.name, func(t *testing.T) {
			ds := tc.newFn()
			cfg, ok := ds.(datasource.DataSourceWithConfigure)
			if !ok {
				t.Fatalf("data source %s does not implement DataSourceWithConfigure", tc.name)
			}
			req := datasource.ConfigureRequest{ProviderData: c}
			resp := datasource.ConfigureResponse{}
			cfg.Configure(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}
