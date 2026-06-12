package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestWarnOnDestroy_AllResources exercises the ModifyPlan hook of every
// resource where the rigor batch wired planhelpers.WarnOnDestroy. The
// hooks are one-liners that delegate to the shared helper (which has
// its own coverage); this file just ensures each per-resource method
// is reached and emits the warning on a destroy plan, closing the
// last 0%-coverage gap on each file.
//
// The helper is `callModifyPlanDelete` from modifyplan_helpers_test.go.
// It feeds the resource a null plan + populated state which is exactly
// the shape `planhelpers.WarnOnDestroy` keys on.
func TestWarnOnDestroy_AllResources(t *testing.T) {
	cases := []struct {
		name string
		r    resource.ResourceWithModifyPlan
	}{
		{"acme_dns_authenticator", NewACMEDNSAuthenticatorResource().(resource.ResourceWithModifyPlan)},
		{"api_key", NewAPIKeyResource().(resource.ResourceWithModifyPlan)},
		{"iscsi_initiator", NewISCSIInitiatorResource().(resource.ResourceWithModifyPlan)},
		{"iscsi_targetextent", NewISCSITargetExtentResource().(resource.ResourceWithModifyPlan)},
		{"kerberos_keytab", NewKerberosKeytabResource().(resource.ResourceWithModifyPlan)},
		{"kerberos_realm", NewKerberosRealmResource().(resource.ResourceWithModifyPlan)},
		{"keychain_credential", NewKeychainCredentialResource().(resource.ResourceWithModifyPlan)},
		{"nvmet_host_subsys", NewNVMetHostSubsysResource().(resource.ResourceWithModifyPlan)},
		{"nvmet_namespace", NewNVMetNamespaceResource().(resource.ResourceWithModifyPlan)},
		{"nvmet_port", NewNVMetPortResource().(resource.ResourceWithModifyPlan)},
		{"nvmet_port_subsys", NewNVMetPortSubsysResource().(resource.ResourceWithModifyPlan)},
		{"nvmet_subsys", NewNVMetSubsysResource().(resource.ResourceWithModifyPlan)},
		{"privilege", NewPrivilegeResource().(resource.ResourceWithModifyPlan)},
		{"vm_device", NewVMDeviceResource().(resource.ResourceWithModifyPlan)},
		{"vmware", NewVMwareResource().(resource.ResourceWithModifyPlan)},
	}
	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {
			resp := callModifyPlanDelete(t, c.r)
			if resp == nil {
				t.Fatal("nil response from ModifyPlan")
			}
			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected error diagnostics: %v", resp.Diagnostics.Errors())
			}
			// The destroy warning is a Warning-level diagnostic. Each
			// resource's ModifyPlan should add at least one warning that
			// mentions "destroy", let the planhelpers package own the
			// exact wording.
			if len(resp.Diagnostics.Warnings()) == 0 {
				t.Errorf("expected WarnOnDestroy to emit a Warning, got none")
			}
		})
	}
}
