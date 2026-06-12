package datasources

// Configure batch: drives every datasource's Configure through the
// three canonical cases — nil ProviderData (silent no-op), wrong-type
// ProviderData (error diagnostic), valid *wsclient.Client (assigns
// client). The datasource twin of the resources-side batch.

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestDataSourceConfigure_Batch(t *testing.T) {
	ctx := context.Background()

	constructors := []func() datasource.DataSource{
		NewAlertServiceDataSource,
		NewAPIKeyDataSource,
		NewAppDataSource,
		NewAppsDataSource,
		NewCatalogDataSource,
		NewCertificateDataSource,
		NewCloudSyncCredentialDataSource,
		NewCronJobDataSource,
		NewDatasetDataSource,
		NewDatasetsDataSource,
		NewDirectoryServicesDataSource,
		NewDiskDataSource,
		NewGroupDataSource,
		NewISCSIExtentDataSource,
		NewISCSIInitiatorDataSource,
		NewISCSIPortalDataSource,
		NewISCSITargetDataSource,
		NewKerberosRealmDataSource,
		NewKeychainCredentialDataSource,
		NewNetworkConfigDataSource,
		NewNetworkInterfaceDataSource,
		NewPoolDataSource,
		NewPoolsDataSource,
		NewPrivilegeDataSource,
		NewServiceDataSource,
		NewShareNFSDataSource,
		NewShareSMBDataSource,
		NewSnapshotTaskDataSource,
		NewSystemDatasetDataSource,
		NewSystemInfoDataSource,
		NewUserDataSource,
		NewVMDataSource,
		NewVMsDataSource,
	}

	for _, ctor := range constructors {
		ds := ctor()
		conf, ok := ds.(datasource.DataSourceWithConfigure)
		if !ok {
			continue
		}
		mdResp := &datasource.MetadataResponse{}
		ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "truenas"}, mdResp)

		t.Run(mdResp.TypeName, func(t *testing.T) {
			// Case 1: nil ProviderData — silent no-op.
			resp1 := &datasource.ConfigureResponse{}
			conf.Configure(ctx, datasource.ConfigureRequest{}, resp1)
			if resp1.Diagnostics.HasError() {
				t.Errorf("Configure(nil) must not error: %v", resp1.Diagnostics)
			}

			// Case 2: wrong-type ProviderData — error diagnostic.
			resp2 := &datasource.ConfigureResponse{}
			conf.Configure(ctx, datasource.ConfigureRequest{ProviderData: "nope"}, resp2)
			if !resp2.Diagnostics.HasError() {
				t.Error("Configure(wrong-type) must error")
			}

			// Case 3: valid client — success. Type-assert only, no dial.
			resp3 := &datasource.ConfigureResponse{}
			conf.Configure(ctx, datasource.ConfigureRequest{ProviderData: &wsclient.Client{}}, resp3)
			if resp3.Diagnostics.HasError() {
				t.Errorf("Configure(valid): %v", resp3.Diagnostics)
			}
		})
	}
}
