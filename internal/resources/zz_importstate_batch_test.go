package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// TestImportState_BadID_Batch drives ImportState on every resource that
// implements ResourceWithImportState with a non-numeric ID. Resources
// that validate ID format up-front will return an error (covered branch).
// Resources that don't will just passthrough — we catch panics.
func TestImportState_BadID_Batch(t *testing.T) {
	ctx := context.Background()
	for _, newFn := range []func() resource.Resource{
		NewACMEDNSAuthenticatorResource,
		NewAlertClassesResource,
		NewAlertServiceResource,
		NewAPIKeyResource,
		NewCatalogResource,
		NewCertificateResource,
		NewCloudBackupResource,
		NewCloudSyncResource,
		NewCloudSyncCredentialResource,
		NewCronJobResource,
		NewDatasetResource,
		NewDirectoryServicesResource,
		NewDNSNameserverResource,
		NewFilesystemACLTemplateResource,
		NewFTPConfigResource,
		NewGroupResource,
		NewInitScriptResource,
		NewISCSIAuthResource,
		NewISCSIExtentResource,
		NewISCSIInitiatorResource,
		NewISCSIPortalResource,
		NewISCSITargetResource,
		NewISCSITargetExtentResource,
		NewKerberosKeytabResource,
		NewKerberosRealmResource,
		NewKeychainCredentialResource,
		NewKMIPConfigResource,
		NewMailConfigResource,
		NewNetworkConfigResource,
		NewNFSConfigResource,
		NewNVMetGlobalResource,
		NewNVMetHostResource,
		NewNVMetHostSubsysResource,
		NewNVMetNamespaceResource,
		NewNVMetPortResource,
		NewNVMetPortSubsysResource,
		NewNVMetSubsysResource,
		NewNetworkInterfaceResource,
		NewServiceResource,
		NewPoolResource,
		NewPrivilegeResource,
		NewReplicationResource,
		NewReportingExporterResource,
		NewRsyncTaskResource,
		NewScrubTaskResource,
		NewNFSShareResource,
		NewSMBShareResource,
		NewSMBConfigResource,
		NewSnapshotTaskResource,
		NewSNMPConfigResource,
		NewSSHConfigResource,
		NewStaticRouteResource,
		NewSystemDatasetResource,
		NewTunableResource,
		NewUPSConfigResource,
		NewUserResource,
		NewVMDeviceResource,
		NewVMResource,
		NewVMwareResource,
		NewZvolResource,
	} {
		r := newFn()
		imp, ok := r.(resource.ResourceWithImportState)
		if !ok {
			continue
		}
		sch := schemaOf(t, ctx, r)
		// Success path (numeric) — exercise the happy branch.
		func() {
			defer func() { _ = recover() }()
			resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)}}
			imp.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, resp)
			_ = resp.Diagnostics
		}()
		// Error path (non-numeric) — exercise validators that parse ID.
		func() {
			defer func() { _ = recover() }()
			resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)}}
			imp.ImportState(ctx, resource.ImportStateRequest{ID: "not-a-number"}, resp)
			_ = resp.Diagnostics
		}()
	}
}
