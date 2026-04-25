package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// schemaCase names a resource constructor and a list of attribute keys that
// must be present in its schema. This file batches Schema + Metadata + New*
// coverage for all resources that don't yet have dedicated mapping tests.
// Running Schema() exercises validators, plan modifiers, timeouts blocks,
// and any inline attribute definitions — substantial lines per resource.
type schemaCase struct {
	name     string
	newFn    func() resource.Resource
	typeName string // expected metadata type suffix (without provider prefix)
	required []string
}

func runSchemaCase(t *testing.T, tc schemaCase) {
	t.Helper()
	ctx := context.Background()
	r := tc.newFn()

	// Metadata
	mdReq := resource.MetadataRequest{ProviderTypeName: "truenas"}
	mdResp := &resource.MetadataResponse{}
	r.Metadata(ctx, mdReq, mdResp)
	if tc.typeName != "" {
		want := "truenas" + tc.typeName
		if mdResp.TypeName != want {
			t.Errorf("Metadata TypeName = %q, want %q", mdResp.TypeName, want)
		}
	}

	// Schema
	schResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schResp)
	if schResp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", schResp.Diagnostics)
	}
	attrs := schResp.Schema.GetAttributes()
	for _, k := range tc.required {
		if _, ok := attrs[k]; !ok {
			t.Errorf("%s: missing required attribute %q", tc.name, k)
		}
	}
}

func TestResourceSchemas_Batch(t *testing.T) {
	cases := []schemaCase{
		{name: "acme_dns_authenticator", newFn: NewACMEDNSAuthenticatorResource, typeName: "_acme_dns_authenticator", required: []string{"id", "name", "authenticator"}},
		{name: "alertclasses", newFn: NewAlertClassesResource, typeName: "_alertclasses", required: []string{"id", "classes"}},
		{name: "app", newFn: NewAppResource, typeName: "_app", required: []string{"id", "app_name"}},
		{name: "catalog", newFn: NewCatalogResource, typeName: "_catalog", required: []string{"id", "label"}},
		{name: "cloudsync_credential", newFn: NewCloudSyncCredentialResource, typeName: "_cloudsync_credential", required: []string{"id", "name", "provider_type"}},
		{name: "directoryservices", newFn: NewDirectoryServicesResource, typeName: "_directoryservices", required: []string{"id"}},
		{name: "filesystem_acl", newFn: NewFilesystemACLResource, typeName: "_filesystem_acl", required: []string{"id", "path"}},
		{name: "filesystem_acl_template", newFn: NewFilesystemACLTemplateResource, typeName: "_filesystem_acl_template", required: []string{"id", "name"}},
		{name: "ftp_config", newFn: NewFTPConfigResource, typeName: "_ftp_config", required: []string{"id"}},
		{name: "iscsi_targetextent", newFn: NewISCSITargetExtentResource, typeName: "_iscsi_targetextent", required: []string{"id", "target", "extent"}},
		{name: "kerberos_keytab", newFn: NewKerberosKeytabResource, typeName: "_kerberos_keytab", required: []string{"id", "name"}},
		{name: "kerberos_realm", newFn: NewKerberosRealmResource, typeName: "_kerberos_realm", required: []string{"id", "realm"}},
		{name: "kmip_config", newFn: NewKMIPConfigResource, typeName: "_kmip_config", required: []string{"id"}},
		{name: "mail_config", newFn: NewMailConfigResource, typeName: "_mail_config", required: []string{"id"}},
		{name: "network_config", newFn: NewNetworkConfigResource, typeName: "_network_config", required: []string{"id"}},
		{name: "nfs_config", newFn: NewNFSConfigResource, typeName: "_nfs_config", required: []string{"id"}},
		{name: "nvmet_global", newFn: NewNVMetGlobalResource, typeName: "_nvmet_global", required: []string{"id"}},
		{name: "nvmet_host_subsys", newFn: NewNVMetHostSubsysResource, typeName: "_nvmet_host_subsys", required: []string{"id", "host_id", "subsys_id"}},
		{name: "nvmet_namespace", newFn: NewNVMetNamespaceResource, typeName: "_nvmet_namespace", required: []string{"id", "subsys_id"}},
		{name: "nvmet_port", newFn: NewNVMetPortResource, typeName: "_nvmet_port", required: []string{"id", "addr_trtype"}},
		{name: "nvmet_port_subsys", newFn: NewNVMetPortSubsysResource, typeName: "_nvmet_port_subsys", required: []string{"id", "port_id", "subsys_id"}},
		{name: "service", newFn: NewServiceResource, typeName: "_service", required: []string{"id", "service"}},
		{name: "smb_config", newFn: NewSMBConfigResource, typeName: "_smb_config", required: []string{"id"}},
		{name: "snmp_config", newFn: NewSNMPConfigResource, typeName: "_snmp_config", required: []string{"id"}},
		{name: "ssh_config", newFn: NewSSHConfigResource, typeName: "_ssh_config", required: []string{"id"}},
		{name: "systemdataset", newFn: NewSystemDatasetResource, typeName: "_systemdataset", required: []string{"id", "pool"}},
		{name: "ups_config", newFn: NewUPSConfigResource, typeName: "_ups_config", required: []string{"id"}},
		{name: "vmware", newFn: NewVMwareResource, typeName: "_vmware", required: []string{"id", "hostname"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runSchemaCase(t, tc)
		})
	}
}
