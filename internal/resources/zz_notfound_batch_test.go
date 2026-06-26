package resources

// Not-found behavioral batch: every id-based resource's Read must take
// the RemoveResource branch (no error diagnostic) and every Delete must
// succeed silently when the server reports the entity is gone. This is
// the unit-level twin of the live _disappears acceptance tests and
// covers the IsNotFound branches in each handler that the happy-path
// CRUD fixtures never reach.

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// newWSNotFoundClient fails every method with the MatchNotFound shape
// TrueNAS emits when an id filter matches zero rows, the canonical
// "deleted out of band" surface.
func newWSNotFoundClient(ctx context.Context, t *testing.T) *wsclient.Client {
	t.Helper()
	ts := wsclient.NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		if strings.HasSuffix(method, ".delete") {
			// Delete on a missing id also surfaces ENOENT.
			return nil, &wsclient.RPCError{
				Code:    wsclient.CodeMethodCallError,
				Message: "Method call error: [ENOENT] None: entity does not exist",
			}
		}
		return nil, &wsclient.RPCError{
			Code:    wsclient.CodeMethodCallError,
			Message: "Method call error: MatchNotFound() (EINVAL)",
		}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c
}

// notFoundCase describes one resource to drive through the batch.
type notFoundCase struct {
	name string
	make func(c *wsclient.Client) resource.Resource
	id   string
}

func TestNotFound_ReadRemovesAndDeleteSucceeds(t *testing.T) {
	ctx := context.Background()

	cases := []notFoundCase{
		{"acme_dns_authenticator", func(c *wsclient.Client) resource.Resource { return &ACMEDNSAuthenticatorResource{client: c} }, "7"},
		{"alert_service", func(c *wsclient.Client) resource.Resource { return &AlertServiceResource{client: c} }, "3"},
		{"api_key", func(c *wsclient.Client) resource.Resource { return &APIKeyResource{client: c} }, "2"},
		{"certificate", func(c *wsclient.Client) resource.Resource { return &CertificateResource{client: c} }, "9"},
		{"cloud_backup", func(c *wsclient.Client) resource.Resource { return &CloudBackupResource{client: c} }, "4"},
		{"cloudsync_credential", func(c *wsclient.Client) resource.Resource { return &CloudSyncCredentialResource{client: c} }, "5"},
		{"cloud_sync", func(c *wsclient.Client) resource.Resource { return &CloudSyncResource{client: c} }, "6"},
		{"cronjob", func(c *wsclient.Client) resource.Resource { return &CronJobResource{client: c} }, "8"},
		{"dataset", func(c *wsclient.Client) resource.Resource { return &DatasetResource{client: c} }, "tank/gone"},
		{"directory", func(c *wsclient.Client) resource.Resource { return &DirectoryResource{client: c} }, "/mnt/tank/gone"},
		// dns_nameserver intentionally absent: it's a virtual slice of the
		// network-config singleton. The singleton always exists, so a
		// not-found from network.configuration.config is a real error the
		// resource must surface, not a remove-from-state case.
		{"filesystem_acl_template", func(c *wsclient.Client) resource.Resource { return &FilesystemACLTemplateResource{client: c} }, "11"},
		{"group", func(c *wsclient.Client) resource.Resource { return &GroupResource{client: c} }, "12"},
		{"init_script", func(c *wsclient.Client) resource.Resource { return &InitScriptResource{client: c} }, "13"},
		{"iscsi_auth", func(c *wsclient.Client) resource.Resource { return &ISCSIAuthResource{client: c} }, "14"},
		{"iscsi_extent", func(c *wsclient.Client) resource.Resource { return &ISCSIExtentResource{client: c} }, "15"},
		{"iscsi_initiator", func(c *wsclient.Client) resource.Resource { return &ISCSIInitiatorResource{client: c} }, "16"},
		{"iscsi_portal", func(c *wsclient.Client) resource.Resource { return &ISCSIPortalResource{client: c} }, "17"},
		{"iscsi_target", func(c *wsclient.Client) resource.Resource { return &ISCSITargetResource{client: c} }, "18"},
		{"iscsi_targetextent", func(c *wsclient.Client) resource.Resource { return &ISCSITargetExtentResource{client: c} }, "19"},
		{"kerberos_keytab", func(c *wsclient.Client) resource.Resource { return &KerberosKeytabResource{client: c} }, "20"},
		{"kerberos_realm", func(c *wsclient.Client) resource.Resource { return &KerberosRealmResource{client: c} }, "21"},
		{"keychain_credential", func(c *wsclient.Client) resource.Resource { return &KeychainCredentialResource{client: c} }, "22"},
		{"nfs_share", func(c *wsclient.Client) resource.Resource { return &NFSShareResource{client: c} }, "23"},
		{"nvmet_host", func(c *wsclient.Client) resource.Resource { return &NVMetHostResource{client: c} }, "24"},
		{"nvmet_host_subsys", func(c *wsclient.Client) resource.Resource { return &NVMetHostSubsysResource{client: c} }, "25"},
		{"nvmet_namespace", func(c *wsclient.Client) resource.Resource { return &NVMetNamespaceResource{client: c} }, "26"},
		{"nvmet_port", func(c *wsclient.Client) resource.Resource { return &NVMetPortResource{client: c} }, "27"},
		{"nvmet_port_subsys", func(c *wsclient.Client) resource.Resource { return &NVMetPortSubsysResource{client: c} }, "28"},
		{"nvmet_subsys", func(c *wsclient.Client) resource.Resource { return &NVMetSubsysResource{client: c} }, "29"},
		{"privilege", func(c *wsclient.Client) resource.Resource { return &PrivilegeResource{client: c} }, "30"},
		{"replication", func(c *wsclient.Client) resource.Resource { return &ReplicationResource{client: c} }, "31"},
		{"reporting_exporter", func(c *wsclient.Client) resource.Resource { return &ReportingExporterResource{client: c} }, "32"},
		{"rsync_task", func(c *wsclient.Client) resource.Resource { return &RsyncTaskResource{client: c} }, "33"},
		{"scrub_task", func(c *wsclient.Client) resource.Resource { return &ScrubTaskResource{client: c} }, "34"},
		{"smb_share", func(c *wsclient.Client) resource.Resource { return &SMBShareResource{client: c} }, "35"},
		{"snapshot_task", func(c *wsclient.Client) resource.Resource { return &SnapshotTaskResource{client: c} }, "36"},
		{"static_route", func(c *wsclient.Client) resource.Resource { return &StaticRouteResource{client: c} }, "37"},
		{"tunable", func(c *wsclient.Client) resource.Resource { return &TunableResource{client: c} }, "38"},
		{"user", func(c *wsclient.Client) resource.Resource { return &UserResource{client: c} }, "39"},
		{"vm", func(c *wsclient.Client) resource.Resource { return &VMResource{client: c} }, "40"},
		{"vm_device", func(c *wsclient.Client) resource.Resource { return &VMDeviceResource{client: c} }, "41"},
		{"vmware", func(c *wsclient.Client) resource.Resource { return &VMwareResource{client: c} }, "42"},
		{"zvol", func(c *wsclient.Client) resource.Resource { return &ZvolResource{client: c} }, "tank/zvol-gone"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newWSNotFoundClient(ctx, t)
			r := tc.make(c)
			sch := schemaOf(t, ctx, r)

			// Read: must take the RemoveResource branch, no diagnostics.
			readState := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str(tc.id)})
			rResp := &resource.ReadResponse{State: readState}
			r.Read(ctx, resource.ReadRequest{State: readState}, rResp)
			if rResp.Diagnostics.HasError() {
				t.Errorf("Read(not-found) produced diagnostics: %v", rResp.Diagnostics)
			}

			// Delete: must be idempotent, no diagnostics.
			dResp := &resource.DeleteResponse{State: readState}
			r.Delete(ctx, resource.DeleteRequest{State: readState}, dResp)
			if dResp.Diagnostics.HasError() {
				t.Errorf("Delete(not-found) produced diagnostics: %v", dResp.Diagnostics)
			}
		})
	}
}
