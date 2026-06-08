package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/sweep"
)

// TestMain is the entry point for `go test -sweep`. It enables sweeper mode
// when -sweep is passed and otherwise behaves like a normal test binary.
// This is the standard Terraform provider sweeper bootstrap pattern.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// sweepCtx is a thin wrapper around internal/sweep.Ctx so existing
// per-resource sweeper functions don't need to rewrite every call site.
func sweepCtx() (context.Context, context.CancelFunc) {
	return sweep.Ctx()
}

// testAccSweeperClient returns a live client.Client for sweeper functions.
// Sweepers are only runnable against a real TrueNAS test VM, so this panics
// if TRUENAS_URL / TRUENAS_API_KEY are missing. That surfaces as a clear
// failure from `go test -sweep=all` rather than silently no-oping.
func testAccSweeperClient() *client.Client {
	c, err := testAccClient()
	if err != nil {
		panic(fmt.Sprintf("sweeper: %v", err))
	}
	return c
}

// sweeperHasAcctestPrefix is a thin wrapper around internal/sweep.HasAcctestPrefix
// so legacy per-resource sweeper callsites don't need to be touched.
func sweeperHasAcctestPrefix(name string) bool {
	return sweep.HasAcctestPrefix(name)
}

// sweeperDatasetIsAcctest is a thin wrapper around internal/sweep.DatasetIsAcctest.
func sweeperDatasetIsAcctest(id string) bool {
	return sweep.DatasetIsAcctest(id)
}

// sweepLog is a thin wrapper around internal/sweep.Log.
func sweepLog(resourceType, action, name string, err error) {
	sweep.Log(resourceType, action, name, err)
}

// sweeperGetList is a thin wrapper around internal/sweep.GetList.
func sweeperGetList(ctx context.Context, c *client.Client, path string, out interface{}) error {
	return sweep.GetList(ctx, c, path, out)
}

// init registers every sweeper defined in this file. Dependency ordering is
// expressed via the Sweeper.Dependencies field rather than registration order
// because the plugin-testing framework iterates sweepers via a Go map (whose
// iteration order is random). Encoding dependencies explicitly guarantees
// that child resources are always swept before their parents.
func init() {
	// ---- Leaf: iscsi_targetextent (must run before target + extent) ----
	resource.AddTestSweepers("truenas_iscsi_targetextent", &resource.Sweeper{
		Name: "truenas_iscsi_targetextent",
		F:    sweepISCSITargetExtents,
	})

	// ---- Leaf: nvmet join tables (must run before host, port, subsys) ----
	resource.AddTestSweepers("truenas_nvmet_host_subsys", &resource.Sweeper{
		Name: "truenas_nvmet_host_subsys",
		F:    sweepNVMetHostSubsys,
	})
	resource.AddTestSweepers("truenas_nvmet_port_subsys", &resource.Sweeper{
		Name: "truenas_nvmet_port_subsys",
		F:    sweepNVMetPortSubsys,
	})

	// ---- Shares and iscsi leaf resources: must run before datasets/zvols --
	resource.AddTestSweepers("truenas_share_nfs", &resource.Sweeper{
		Name: "truenas_share_nfs",
		F:    sweepNFSShares,
	})
	resource.AddTestSweepers("truenas_share_smb", &resource.Sweeper{
		Name: "truenas_share_smb",
		F:    sweepSMBShares,
	})
	resource.AddTestSweepers("truenas_nvmet_namespace", &resource.Sweeper{
		Name:         "truenas_nvmet_namespace",
		F:            sweepNVMetNamespaces,
		Dependencies: []string{"truenas_nvmet_host_subsys", "truenas_nvmet_port_subsys"},
	})
	resource.AddTestSweepers("truenas_iscsi_extent", &resource.Sweeper{
		Name:         "truenas_iscsi_extent",
		F:            sweepISCSIExtents,
		Dependencies: []string{"truenas_iscsi_targetextent"},
	})
	resource.AddTestSweepers("truenas_iscsi_target", &resource.Sweeper{
		Name:         "truenas_iscsi_target",
		F:            sweepISCSITargets,
		Dependencies: []string{"truenas_iscsi_targetextent"},
	})
	resource.AddTestSweepers("truenas_iscsi_portal", &resource.Sweeper{
		Name:         "truenas_iscsi_portal",
		F:            sweepISCSIPortals,
		Dependencies: []string{"truenas_iscsi_target"},
	})
	resource.AddTestSweepers("truenas_iscsi_initiator", &resource.Sweeper{
		Name:         "truenas_iscsi_initiator",
		F:            sweepISCSIInitiators,
		Dependencies: []string{"truenas_iscsi_target"},
	})
	resource.AddTestSweepers("truenas_iscsi_auth", &resource.Sweeper{
		Name:         "truenas_iscsi_auth",
		F:            sweepISCSIAuths,
		Dependencies: []string{"truenas_iscsi_target"},
	})

	// ---- VM devices before VMs ------------------------------------------
	resource.AddTestSweepers("truenas_vm_device", &resource.Sweeper{
		Name: "truenas_vm_device",
		F:    sweepVMDevices,
	})
	resource.AddTestSweepers("truenas_vm", &resource.Sweeper{
		Name:         "truenas_vm",
		F:            sweepVMs,
		Dependencies: []string{"truenas_vm_device"},
	})

	// ---- Tasks that may reference datasets ------------------------------
	resource.AddTestSweepers("truenas_cronjob", &resource.Sweeper{
		Name: "truenas_cronjob",
		F:    sweepCronJobs,
	})
	resource.AddTestSweepers("truenas_init_script", &resource.Sweeper{
		Name: "truenas_init_script",
		F:    sweepInitScripts,
	})
	resource.AddTestSweepers("truenas_rsync_task", &resource.Sweeper{
		Name: "truenas_rsync_task",
		F:    sweepRsyncTasks,
	})
	resource.AddTestSweepers("truenas_scrub_task", &resource.Sweeper{
		Name: "truenas_scrub_task",
		F:    sweepScrubTasks,
	})
	resource.AddTestSweepers("truenas_snapshot_task", &resource.Sweeper{
		Name: "truenas_snapshot_task",
		F:    sweepSnapshotTasks,
	})
	resource.AddTestSweepers("truenas_replication", &resource.Sweeper{
		Name: "truenas_replication",
		F:    sweepReplications,
	})

	// ---- Storage: datasets and zvols (LAST among storage) ----------------
	resource.AddTestSweepers("truenas_zvol", &resource.Sweeper{
		Name: "truenas_zvol",
		F:    sweepZvols,
		Dependencies: []string{
			"truenas_share_nfs",
			"truenas_share_smb",
			"truenas_iscsi_extent",
			"truenas_nvmet_namespace",
			"truenas_snapshot_task",
			"truenas_replication",
			"truenas_vm",
		},
	})
	resource.AddTestSweepers("truenas_dataset", &resource.Sweeper{
		Name: "truenas_dataset",
		F:    sweepDatasets,
		Dependencies: []string{
			"truenas_zvol",
			"truenas_share_nfs",
			"truenas_share_smb",
			"truenas_iscsi_extent",
			"truenas_nvmet_namespace",
			"truenas_snapshot_task",
			"truenas_replication",
			"truenas_vm",
		},
	})

	// ---- Identity: user/group (order between them doesn't matter) -------
	resource.AddTestSweepers("truenas_user", &resource.Sweeper{
		Name: "truenas_user",
		F:    sweepUsers,
	})
	resource.AddTestSweepers("truenas_group", &resource.Sweeper{
		Name: "truenas_group",
		F:    sweepGroups,
	})

	// ---- Independent resources (no ordering constraints) ----------------
	resource.AddTestSweepers("truenas_static_route", &resource.Sweeper{
		Name: "truenas_static_route",
		F:    sweepStaticRoutes,
	})
	resource.AddTestSweepers("truenas_alert_service", &resource.Sweeper{
		Name: "truenas_alert_service",
		F:    sweepAlertServices,
	})
	resource.AddTestSweepers("truenas_tunable", &resource.Sweeper{
		Name: "truenas_tunable",
		F:    sweepTunables,
	})
	resource.AddTestSweepers("truenas_reporting_exporter", &resource.Sweeper{
		Name: "truenas_reporting_exporter",
		F:    sweepReportingExporters,
	})
	resource.AddTestSweepers("truenas_cloudsync_credential", &resource.Sweeper{
		Name: "truenas_cloudsync_credential",
		F:    sweepCloudSyncCredentials,
	})
	resource.AddTestSweepers("truenas_privilege", &resource.Sweeper{
		Name: "truenas_privilege",
		F:    sweepPrivileges,
	})
	resource.AddTestSweepers("truenas_api_key", &resource.Sweeper{
		Name: "truenas_api_key",
		F:    sweepAPIKeys,
	})
	resource.AddTestSweepers("truenas_keychain_credential", &resource.Sweeper{
		Name: "truenas_keychain_credential",
		F:    sweepKeychainCredentials,
	})
	resource.AddTestSweepers("truenas_certificate", &resource.Sweeper{
		Name: "truenas_certificate",
		F:    sweepCertificates,
	})
	resource.AddTestSweepers("truenas_kerberos_realm", &resource.Sweeper{
		Name: "truenas_kerberos_realm",
		F:    sweepKerberosRealms,
	})
	resource.AddTestSweepers("truenas_kerberos_keytab", &resource.Sweeper{
		Name: "truenas_kerberos_keytab",
		F:    sweepKerberosKeytabs,
	})
	resource.AddTestSweepers("truenas_filesystem_acl_template", &resource.Sweeper{
		Name: "truenas_filesystem_acl_template",
		F:    sweepFilesystemACLTemplates,
	})
	resource.AddTestSweepers("truenas_acme_dns_authenticator", &resource.Sweeper{
		Name: "truenas_acme_dns_authenticator",
		F:    sweepACMEDNSAuthenticators,
	})
	resource.AddTestSweepers("truenas_app", &resource.Sweeper{
		Name: "truenas_app",
		F:    sweepApps,
	})
	resource.AddTestSweepers("truenas_cloud_backup", &resource.Sweeper{
		Name: "truenas_cloud_backup",
		F:    sweepCloudBackups,
	})
	resource.AddTestSweepers("truenas_cloud_sync", &resource.Sweeper{
		Name: "truenas_cloud_sync",
		F:    sweepCloudSyncs,
	})
	resource.AddTestSweepers("truenas_vmware", &resource.Sweeper{
		Name: "truenas_vmware",
		F:    sweepVMwareIntegrations,
	})

	// ---- NVMet base resources: swept AFTER namespace + join tables ------
	resource.AddTestSweepers("truenas_nvmet_port", &resource.Sweeper{
		Name:         "truenas_nvmet_port",
		F:            sweepNVMetPorts,
		Dependencies: []string{"truenas_nvmet_namespace", "truenas_nvmet_port_subsys"},
	})
	resource.AddTestSweepers("truenas_nvmet_subsys", &resource.Sweeper{
		Name:         "truenas_nvmet_subsys",
		F:            sweepNVMetSubsys,
		Dependencies: []string{"truenas_nvmet_namespace", "truenas_nvmet_host_subsys", "truenas_nvmet_port_subsys"},
	})
	resource.AddTestSweepers("truenas_nvmet_host", &resource.Sweeper{
		Name:         "truenas_nvmet_host",
		F:            sweepNVMetHosts,
		Dependencies: []string{"truenas_nvmet_host_subsys"},
	})
}

// =====================================================================
// Sweeper functions
// =====================================================================

// Each sweeper is a standalone function with the signature
// `func(region string) error`. We ignore `region` because TrueNAS is a
// single-region product. Every sweeper follows the same shape:
//   1. Build a context + client
//   2. List all resources of the target type
//   3. Filter to fixtures (strict prefix match)
//   4. Delete each fixture, logging success/failure
//   5. Continue past individual failures, return only fatal errors

func sweepUsers(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	users, err := c.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_user: listing: %w", err)
	}
	for _, u := range users {
		if u.Builtin || !sweeperHasAcctestPrefix(u.Username) {
			continue
		}
		err := c.DeleteUser(ctx, u.ID)
		sweepLog("truenas_user", "delete", u.Username, err)
	}
	return nil
}

func sweepGroups(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	groups, err := c.ListGroups(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_group: listing: %w", err)
	}
	for _, g := range groups {
		if g.Builtin || !sweeperHasAcctestPrefix(g.Name) {
			continue
		}
		err := c.DeleteGroup(ctx, g.ID)
		sweepLog("truenas_group", "delete", g.Name, err)
	}
	return nil
}

func sweepDatasets(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	datasets, err := c.ListDatasets(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_dataset: listing: %w", err)
	}
	for _, d := range datasets {
		if d.Type == "VOLUME" {
			continue // handled by sweepZvols
		}
		if !sweeperDatasetIsAcctest(d.ID) {
			continue
		}
		err := c.DeleteDataset(ctx, d.ID)
		sweepLog("truenas_dataset", "delete", d.ID, err)
	}
	return nil
}

func sweepZvols(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	datasets, err := c.ListDatasets(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_zvol: listing: %w", err)
	}
	for _, d := range datasets {
		if d.Type != "VOLUME" {
			continue
		}
		if !sweeperDatasetIsAcctest(d.ID) {
			continue
		}
		err := c.DeleteZvol(ctx, d.ID)
		sweepLog("truenas_zvol", "delete", d.ID, err)
	}
	return nil
}

func sweepNFSShares(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	shares, err := c.ListNFSShares(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_share_nfs: listing: %w", err)
	}
	for _, s := range shares {
		// NFS shares have no name — match by comment or by dataset path.
		if !sweeperHasAcctestPrefix(s.Comment) && !sweeperDatasetIsAcctest(s.Path) {
			continue
		}
		err := c.DeleteNFSShare(ctx, s.ID)
		sweepLog("truenas_share_nfs", "delete", s.Path, err)
	}
	return nil
}

func sweepSMBShares(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	shares, err := c.ListSMBShares(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_share_smb: listing: %w", err)
	}
	for _, s := range shares {
		if !sweeperHasAcctestPrefix(s.Name) && !sweeperDatasetIsAcctest(s.Path) {
			continue
		}
		err := c.DeleteSMBShare(ctx, s.ID)
		sweepLog("truenas_share_smb", "delete", s.Name, err)
	}
	return nil
}

// iscsi resources: none have ListXxx methods on the client, so we call
// the collection GET endpoint directly via sweeperGetList.

func sweepISCSIPortals(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var portals []client.ISCSIPortal
	if err := sweeperGetList(ctx, c, "/iscsi/portal", &portals); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_portal: listing: %w", err)
	}
	for _, p := range portals {
		if !sweeperHasAcctestPrefix(p.Comment) {
			continue
		}
		err := c.DeleteISCSIPortal(ctx, p.ID)
		sweepLog("truenas_iscsi_portal", "delete", p.Comment, err)
	}
	return nil
}

func sweepISCSITargets(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var targets []client.ISCSITarget
	if err := sweeperGetList(ctx, c, "/iscsi/target", &targets); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_target: listing: %w", err)
	}
	for _, t := range targets {
		if !sweeperHasAcctestPrefix(t.Name) {
			continue
		}
		err := c.DeleteISCSITarget(ctx, t.ID)
		sweepLog("truenas_iscsi_target", "delete", t.Name, err)
	}
	return nil
}

func sweepISCSIExtents(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var extents []client.ISCSIExtent
	if err := sweeperGetList(ctx, c, "/iscsi/extent", &extents); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_extent: listing: %w", err)
	}
	for _, e := range extents {
		if !sweeperHasAcctestPrefix(e.Name) {
			continue
		}
		err := c.DeleteISCSIExtent(ctx, e.ID)
		sweepLog("truenas_iscsi_extent", "delete", e.Name, err)
	}
	return nil
}

func sweepISCSIInitiators(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var inits []client.ISCSIInitiator
	if err := sweeperGetList(ctx, c, "/iscsi/initiator", &inits); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_initiator: listing: %w", err)
	}
	for _, i := range inits {
		if !sweeperHasAcctestPrefix(i.Comment) {
			continue
		}
		err := c.DeleteISCSIInitiator(ctx, i.ID)
		sweepLog("truenas_iscsi_initiator", "delete", i.Comment, err)
	}
	return nil
}

// sweepISCSIAuths cannot reliably identify test fixtures (ISCSIAuth has only
// Tag/User/Secret — all opaque numbers/strings). We sweep entries whose User
// matches the acctest prefix to stay safe.
func sweepISCSIAuths(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var auths []client.ISCSIAuth
	if err := sweeperGetList(ctx, c, "/iscsi/auth", &auths); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_auth: listing: %w", err)
	}
	for _, a := range auths {
		if !sweeperHasAcctestPrefix(a.User) {
			continue
		}
		err := c.DeleteISCSIAuth(ctx, a.ID)
		sweepLog("truenas_iscsi_auth", "delete", a.User, err)
	}
	return nil
}

// sweepISCSITargetExtents: targetextent has no name, so we can only sweep
// associations whose Target OR Extent has already been marked for sweeping.
// Safest approach: delete any targetextent whose referenced target name
// looks like an acctest fixture.
func sweepISCSITargetExtents(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var targets []client.ISCSITarget
	if err := sweeperGetList(ctx, c, "/iscsi/target", &targets); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_targetextent: listing targets: %w", err)
	}
	acctestTargets := make(map[int]string, len(targets))
	for _, t := range targets {
		if sweeperHasAcctestPrefix(t.Name) {
			acctestTargets[t.ID] = t.Name
		}
	}

	var extents []client.ISCSIExtent
	if err := sweeperGetList(ctx, c, "/iscsi/extent", &extents); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_targetextent: listing extents: %w", err)
	}
	acctestExtents := make(map[int]string, len(extents))
	for _, e := range extents {
		if sweeperHasAcctestPrefix(e.Name) {
			acctestExtents[e.ID] = e.Name
		}
	}

	var tes []client.ISCSITargetExtent
	if err := sweeperGetList(ctx, c, "/iscsi/targetextent", &tes); err != nil {
		return fmt.Errorf("sweeping truenas_iscsi_targetextent: listing: %w", err)
	}
	for _, te := range tes {
		_, tOK := acctestTargets[te.Target]
		_, eOK := acctestExtents[te.Extent]
		if !tOK && !eOK {
			continue
		}
		label := fmt.Sprintf("target=%d extent=%d", te.Target, te.Extent)
		err := c.DeleteISCSITargetExtent(ctx, te.ID)
		sweepLog("truenas_iscsi_targetextent", "delete", label, err)
	}
	return nil
}

// snapshot/cron/init/rsync/scrub/replication tasks ---------------------------

func sweepSnapshotTasks(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var tasks []client.SnapshotTask
	if err := sweeperGetList(ctx, c, "/pool/snapshottask", &tasks); err != nil {
		return fmt.Errorf("sweeping truenas_snapshot_task: listing: %w", err)
	}
	for _, t := range tasks {
		if !sweeperDatasetIsAcctest(t.Dataset) {
			continue
		}
		err := c.DeleteSnapshotTask(ctx, t.ID)
		sweepLog("truenas_snapshot_task", "delete", t.Dataset, err)
	}
	return nil
}

func sweepCronJobs(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	jobs, err := c.ListCronJobs(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_cronjob: listing: %w", err)
	}
	for _, j := range jobs {
		if !sweeperHasAcctestPrefix(j.Description) {
			continue
		}
		err := c.DeleteCronJob(ctx, j.ID)
		sweepLog("truenas_cronjob", "delete", j.Description, err)
	}
	return nil
}

func sweepInitScripts(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var scripts []client.InitScript
	if err := sweeperGetList(ctx, c, "/initshutdownscript", &scripts); err != nil {
		return fmt.Errorf("sweeping truenas_init_script: listing: %w", err)
	}
	for _, s := range scripts {
		if !sweeperHasAcctestPrefix(s.Comment) {
			continue
		}
		err := c.DeleteInitScript(ctx, s.ID)
		sweepLog("truenas_init_script", "delete", s.Comment, err)
	}
	return nil
}

func sweepRsyncTasks(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var tasks []client.RsyncTask
	if err := sweeperGetList(ctx, c, "/rsynctask", &tasks); err != nil {
		return fmt.Errorf("sweeping truenas_rsync_task: listing: %w", err)
	}
	for _, t := range tasks {
		if !sweeperHasAcctestPrefix(t.Desc) && !sweeperDatasetIsAcctest(t.Path) {
			continue
		}
		err := c.DeleteRsyncTask(ctx, t.ID)
		sweepLog("truenas_rsync_task", "delete", t.Desc, err)
	}
	return nil
}

func sweepScrubTasks(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var tasks []client.ScrubTask
	if err := sweeperGetList(ctx, c, "/pool/scrub", &tasks); err != nil {
		return fmt.Errorf("sweeping truenas_scrub_task: listing: %w", err)
	}
	for _, t := range tasks {
		if !sweeperHasAcctestPrefix(t.Description) {
			continue
		}
		err := c.DeleteScrubTask(ctx, t.ID)
		sweepLog("truenas_scrub_task", "delete", t.Description, err)
	}
	return nil
}

func sweepReplications(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var reps []client.Replication
	if err := sweeperGetList(ctx, c, "/replication", &reps); err != nil {
		return fmt.Errorf("sweeping truenas_replication: listing: %w", err)
	}
	for _, r := range reps {
		if !sweeperHasAcctestPrefix(r.Name) {
			continue
		}
		err := c.DeleteReplication(ctx, r.ID)
		sweepLog("truenas_replication", "delete", r.Name, err)
	}
	return nil
}

// network / global / aux --------------------------------------------------

func sweepStaticRoutes(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var routes []client.StaticRoute
	if err := sweeperGetList(ctx, c, "/staticroute", &routes); err != nil {
		return fmt.Errorf("sweeping truenas_static_route: listing: %w", err)
	}
	for _, r := range routes {
		if !sweeperHasAcctestPrefix(r.Description) {
			continue
		}
		err := c.DeleteStaticRoute(ctx, r.ID)
		sweepLog("truenas_static_route", "delete", r.Description, err)
	}
	return nil
}

func sweepAlertServices(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var svcs []client.AlertService
	if err := sweeperGetList(ctx, c, "/alertservice", &svcs); err != nil {
		return fmt.Errorf("sweeping truenas_alert_service: listing: %w", err)
	}
	for _, s := range svcs {
		if !sweeperHasAcctestPrefix(s.Name) {
			continue
		}
		err := c.DeleteAlertService(ctx, s.ID)
		sweepLog("truenas_alert_service", "delete", s.Name, err)
	}
	return nil
}

func sweepTunables(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	tunables, err := c.ListTunables(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_tunable: listing: %w", err)
	}
	for _, t := range tunables {
		// Tunables cannot be named with arbitrary prefixes (Var must be a
		// valid sysctl name), so match on Comment instead.
		if !sweeperHasAcctestPrefix(t.Comment) {
			continue
		}
		err := c.DeleteTunable(ctx, t.ID)
		sweepLog("truenas_tunable", "delete", t.Var, err)
	}
	return nil
}

func sweepReportingExporters(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var exps []client.ReportingExporter
	if err := sweeperGetList(ctx, c, "/reporting/exporters", &exps); err != nil {
		return fmt.Errorf("sweeping truenas_reporting_exporter: listing: %w", err)
	}
	for _, e := range exps {
		if !sweeperHasAcctestPrefix(e.Name) {
			continue
		}
		err := c.DeleteReportingExporter(ctx, e.ID)
		sweepLog("truenas_reporting_exporter", "delete", e.Name, err)
	}
	return nil
}

func sweepCloudSyncCredentials(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	creds, err := c.ListCloudSyncCredentials(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_cloudsync_credential: listing: %w", err)
	}
	for _, cr := range creds {
		if !sweeperHasAcctestPrefix(cr.Name) {
			continue
		}
		err := c.DeleteCloudSyncCredential(ctx, cr.ID)
		sweepLog("truenas_cloudsync_credential", "delete", cr.Name, err)
	}
	return nil
}

func sweepPrivileges(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	privs, err := c.ListPrivileges(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_privilege: listing: %w", err)
	}
	for _, p := range privs {
		if p.BuiltinName != nil && *p.BuiltinName != "" {
			continue
		}
		if !sweeperHasAcctestPrefix(p.Name) {
			continue
		}
		err := c.DeletePrivilege(ctx, p.ID)
		sweepLog("truenas_privilege", "delete", p.Name, err)
	}
	return nil
}

func sweepAPIKeys(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var keys []client.APIKey
	if err := sweeperGetList(ctx, c, "/api_key", &keys); err != nil {
		return fmt.Errorf("sweeping truenas_api_key: listing: %w", err)
	}
	for _, k := range keys {
		if !sweeperHasAcctestPrefix(k.Name) {
			continue
		}
		err := c.DeleteAPIKey(ctx, k.ID)
		sweepLog("truenas_api_key", "delete", k.Name, err)
	}
	return nil
}

func sweepKeychainCredentials(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var creds []client.KeychainCredential
	if err := sweeperGetList(ctx, c, "/keychaincredential", &creds); err != nil {
		return fmt.Errorf("sweeping truenas_keychain_credential: listing: %w", err)
	}
	for _, cr := range creds {
		if !sweeperHasAcctestPrefix(cr.Name) {
			continue
		}
		err := c.DeleteKeychainCredential(ctx, cr.ID)
		sweepLog("truenas_keychain_credential", "delete", cr.Name, err)
	}
	return nil
}

func sweepCertificates(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	certs, err := c.ListCertificates(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_certificate: listing: %w", err)
	}
	for _, ct := range certs {
		if !sweeperHasAcctestPrefix(ct.Name) {
			continue
		}
		err := c.DeleteCertificate(ctx, ct.ID)
		sweepLog("truenas_certificate", "delete", ct.Name, err)
	}
	return nil
}

func sweepKerberosRealms(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	realms, err := c.ListKerberosRealms(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_kerberos_realm: listing: %w", err)
	}
	for _, r := range realms {
		if !sweeperHasAcctestPrefix(r.Realm) {
			continue
		}
		err := c.DeleteKerberosRealm(ctx, r.ID)
		sweepLog("truenas_kerberos_realm", "delete", r.Realm, err)
	}
	return nil
}

func sweepKerberosKeytabs(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var keytabs []client.KerberosKeytab
	if err := sweeperGetList(ctx, c, "/kerberos/keytab", &keytabs); err != nil {
		return fmt.Errorf("sweeping truenas_kerberos_keytab: listing: %w", err)
	}
	for _, k := range keytabs {
		if !sweeperHasAcctestPrefix(k.Name) {
			continue
		}
		err := c.DeleteKerberosKeytab(ctx, k.ID)
		sweepLog("truenas_kerberos_keytab", "delete", k.Name, err)
	}
	return nil
}

func sweepFilesystemACLTemplates(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var tpls []client.FilesystemACLTemplate
	if err := sweeperGetList(ctx, c, "/filesystem/acltemplate", &tpls); err != nil {
		return fmt.Errorf("sweeping truenas_filesystem_acl_template: listing: %w", err)
	}
	for _, t := range tpls {
		if t.Builtin || !sweeperHasAcctestPrefix(t.Name) {
			continue
		}
		err := c.DeleteFilesystemACLTemplate(ctx, t.ID)
		sweepLog("truenas_filesystem_acl_template", "delete", t.Name, err)
	}
	return nil
}

// NVMe-oF --------------------------------------------------------------

// sweepNVMetHostSubsys removes host<->subsys join rows where either side
// is an acctest fixture. Running before host+subsys sweeps ensures the
// parents can be deleted cleanly.
func sweepNVMetHostSubsys(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var hosts []client.NVMetHost
	if err := sweeperGetList(ctx, c, "/nvmet/host", &hosts); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_host_subsys: listing hosts: %w", err)
	}
	acctHosts := make(map[int]bool, len(hosts))
	for _, h := range hosts {
		if strings.Contains(strings.ToLower(h.Hostnqn), "nqn.2014-08") ||
			sweeperHasAcctestPrefix(h.Hostnqn) {
			acctHosts[h.ID] = true
		}
	}

	var subs []client.NVMetSubsys
	if err := sweeperGetList(ctx, c, "/nvmet/subsys", &subs); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_host_subsys: listing subsys: %w", err)
	}
	acctSubs := make(map[int]bool, len(subs))
	for _, s := range subs {
		if sweeperHasAcctestPrefix(s.Name) {
			acctSubs[s.ID] = true
		}
	}

	type hostSubsysRow struct {
		ID     int `json:"id"`
		Host   int `json:"host"`
		Subsys int `json:"subsys"`
	}
	var rows []hostSubsysRow
	if err := sweeperGetList(ctx, c, "/nvmet/host_subsys", &rows); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_host_subsys: listing: %w", err)
	}
	for _, r := range rows {
		if !acctHosts[r.Host] && !acctSubs[r.Subsys] {
			continue
		}
		err := c.DeleteNVMetHostSubsys(ctx, r.ID)
		sweepLog("truenas_nvmet_host_subsys", "delete",
			fmt.Sprintf("host=%d subsys=%d", r.Host, r.Subsys), err)
	}
	return nil
}

func sweepNVMetPortSubsys(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var ports []client.NVMetPort
	if err := sweeperGetList(ctx, c, "/nvmet/port", &ports); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_port_subsys: listing ports: %w", err)
	}
	acctPorts := make(map[int]bool, len(ports))
	for _, p := range ports {
		if sweeperHasAcctestPrefix(p.AddrTraddr) {
			acctPorts[p.ID] = true
		}
	}

	var subs []client.NVMetSubsys
	if err := sweeperGetList(ctx, c, "/nvmet/subsys", &subs); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_port_subsys: listing subsys: %w", err)
	}
	acctSubs := make(map[int]bool, len(subs))
	for _, s := range subs {
		if sweeperHasAcctestPrefix(s.Name) {
			acctSubs[s.ID] = true
		}
	}

	type portSubsysRow struct {
		ID     int `json:"id"`
		Port   int `json:"port"`
		Subsys int `json:"subsys"`
	}
	var rows []portSubsysRow
	if err := sweeperGetList(ctx, c, "/nvmet/port_subsys", &rows); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_port_subsys: listing: %w", err)
	}
	for _, r := range rows {
		if !acctPorts[r.Port] && !acctSubs[r.Subsys] {
			continue
		}
		err := c.DeleteNVMetPortSubsys(ctx, r.ID)
		sweepLog("truenas_nvmet_port_subsys", "delete",
			fmt.Sprintf("port=%d subsys=%d", r.Port, r.Subsys), err)
	}
	return nil
}

func sweepNVMetNamespaces(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	// Build the acctest subsys set first so we can match namespaces.
	var subs []client.NVMetSubsys
	if err := sweeperGetList(ctx, c, "/nvmet/subsys", &subs); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_namespace: listing subsys: %w", err)
	}
	acctSubs := make(map[int]bool, len(subs))
	for _, s := range subs {
		if sweeperHasAcctestPrefix(s.Name) {
			acctSubs[s.ID] = true
		}
	}

	var namespaces []client.NVMetNamespace
	if err := sweeperGetList(ctx, c, "/nvmet/namespace", &namespaces); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_namespace: listing: %w", err)
	}
	for _, n := range namespaces {
		subsysID := n.SubsysID
		if n.Subsys != nil && subsysID == 0 {
			subsysID = n.Subsys.ID
		}
		if !acctSubs[subsysID] && !sweeperDatasetIsAcctest(n.DevicePath) {
			continue
		}
		err := c.DeleteNVMetNamespace(ctx, n.ID)
		sweepLog("truenas_nvmet_namespace", "delete", n.DevicePath, err)
	}
	return nil
}

func sweepNVMetPorts(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var ports []client.NVMetPort
	if err := sweeperGetList(ctx, c, "/nvmet/port", &ports); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_port: listing: %w", err)
	}
	for _, p := range ports {
		if !sweeperHasAcctestPrefix(p.AddrTraddr) {
			continue
		}
		err := c.DeleteNVMetPort(ctx, p.ID)
		sweepLog("truenas_nvmet_port", "delete", p.AddrTraddr, err)
	}
	return nil
}

func sweepNVMetSubsys(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var subs []client.NVMetSubsys
	if err := sweeperGetList(ctx, c, "/nvmet/subsys", &subs); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_subsys: listing: %w", err)
	}
	for _, s := range subs {
		if !sweeperHasAcctestPrefix(s.Name) {
			continue
		}
		err := c.DeleteNVMetSubsys(ctx, s.ID)
		sweepLog("truenas_nvmet_subsys", "delete", s.Name, err)
	}
	return nil
}

func sweepNVMetHosts(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var hosts []client.NVMetHost
	if err := sweeperGetList(ctx, c, "/nvmet/host", &hosts); err != nil {
		return fmt.Errorf("sweeping truenas_nvmet_host: listing: %w", err)
	}
	for _, h := range hosts {
		// Test fixtures use either an acctest prefix on the NQN or the
		// standard "nqn.2014-08" prefix (our helper uses that for NVMe-oF).
		low := strings.ToLower(h.Hostnqn)
		if !sweeperHasAcctestPrefix(h.Hostnqn) && !strings.Contains(low, "acct") {
			continue
		}
		err := c.DeleteNVMetHost(ctx, h.ID)
		sweepLog("truenas_nvmet_host", "delete", h.Hostnqn, err)
	}
	return nil
}

// VMs ------------------------------------------------------------------

func sweepVMs(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	vms, err := c.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_vm: listing: %w", err)
	}
	for _, v := range vms {
		if !sweeperHasAcctestPrefix(v.Name) {
			continue
		}
		// Force + leave zvols (sweepZvols will handle them).
		err := c.DeleteVM(ctx, v.ID, &client.VMDeleteOptions{Force: true, Zvols: false})
		sweepLog("truenas_vm", "delete", v.Name, err)
	}
	return nil
}

// sweepVMDevices removes devices that belong to a VM whose name looks like
// an acctest fixture. Devices have no user-visible name so parent matching
// is the only safe approach.
func sweepVMDevices(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	vms, err := c.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_vm_device: listing VMs: %w", err)
	}
	acctVMs := make(map[int]string, len(vms))
	for _, v := range vms {
		if sweeperHasAcctestPrefix(v.Name) {
			acctVMs[v.ID] = v.Name
		}
	}

	var devs []client.VMDevice
	if err := sweeperGetList(ctx, c, "/vm/device", &devs); err != nil {
		return fmt.Errorf("sweeping truenas_vm_device: listing: %w", err)
	}
	for _, d := range devs {
		name, ok := acctVMs[d.VM]
		if !ok {
			continue
		}
		err := c.DeleteVMDevice(ctx, d.ID)
		sweepLog("truenas_vm_device", "delete",
			fmt.Sprintf("vm=%s device=%d", name, d.ID), err)
	}
	return nil
}

// ---- Pending-resource sweepers (Phase B) --------------------------------
//
// These five sweepers were the "pending" entries in resourceSweeperExclusions
// before Phase B. Each uses sweeperGetList because the client does not
// expose a typed ListXxx method — the collection GET is fine because every
// resource type here returns a JSON array at its collection URL.

func sweepACMEDNSAuthenticators(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var auths []client.ACMEDNSAuthenticator
	if err := sweeperGetList(ctx, c, "/acme/dns/authenticator", &auths); err != nil {
		return fmt.Errorf("sweeping truenas_acme_dns_authenticator: listing: %w", err)
	}
	for _, a := range auths {
		if !sweeperHasAcctestPrefix(a.Name) {
			continue
		}
		err := c.DeleteACMEDNSAuthenticator(ctx, a.ID)
		sweepLog("truenas_acme_dns_authenticator", "delete", a.Name, err)
	}
	return nil
}

func sweepApps(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	apps, err := c.ListApps(ctx)
	if err != nil {
		return fmt.Errorf("sweeping truenas_app: listing: %w", err)
	}
	for _, a := range apps {
		// App.Name is the release name (unique per cluster), which is
		// what test fixtures generate via randomName(). Match strictly
		// on the acctest prefix — never sweep user-created apps.
		if !sweeperHasAcctestPrefix(a.Name) {
			continue
		}
		// DeleteApp accepts an optional delete-data flag; always pass
		// the default zero value here so we do not accidentally wipe
		// an operator's PVCs if an acctest fixture name collides.
		err := c.DeleteApp(ctx, a.ID, nil)
		sweepLog("truenas_app", "delete", a.Name, err)
	}
	return nil
}

func sweepCloudBackups(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var backups []client.CloudBackup
	if err := sweeperGetList(ctx, c, "/cloud_backup", &backups); err != nil {
		return fmt.Errorf("sweeping truenas_cloud_backup: listing: %w", err)
	}
	for _, b := range backups {
		if !sweeperHasAcctestPrefix(b.Description) {
			continue
		}
		err := c.DeleteCloudBackup(ctx, b.ID)
		sweepLog("truenas_cloud_backup", "delete", b.Description, err)
	}
	return nil
}

func sweepCloudSyncs(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var syncs []client.CloudSync
	if err := sweeperGetList(ctx, c, "/cloudsync", &syncs); err != nil {
		return fmt.Errorf("sweeping truenas_cloud_sync: listing: %w", err)
	}
	for _, s := range syncs {
		if !sweeperHasAcctestPrefix(s.Description) {
			continue
		}
		err := c.DeleteCloudSync(ctx, s.ID)
		sweepLog("truenas_cloud_sync", "delete", s.Description, err)
	}
	return nil
}

func sweepVMwareIntegrations(_ string) error {
	ctx, cancel := sweepCtx()
	defer cancel()
	c := testAccSweeperClient()

	var vms []client.VMware
	if err := sweeperGetList(ctx, c, "/vmware", &vms); err != nil {
		return fmt.Errorf("sweeping truenas_vmware: listing: %w", err)
	}
	for _, v := range vms {
		// VMware integrations identify by hostname + datastore. Test
		// fixtures set the hostname to an acctest-prefixed value.
		// Real VMware integrations have real hostnames, so the prefix
		// match is strong enough to prevent accidental sweeping.
		if !sweeperHasAcctestPrefix(v.Hostname) {
			continue
		}
		err := c.DeleteVMware(ctx, v.ID)
		sweepLog("truenas_vmware",
			"delete", fmt.Sprintf("%s@%s", v.Datastore, v.Hostname), err)
	}
	return nil
}
