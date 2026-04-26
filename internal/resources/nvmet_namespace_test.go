package resources_test

import (
	"testing"
)

// TestAccNVMetNamespace_basic is intentionally skipped because creating an
// NVMe-oF namespace requires a live ZVOL or backing file on the target
// TrueNAS, which cannot be reliably provisioned and torn down in a generic
// acceptance environment. The resource is exercised by the NVMet integration
// suite when run by hand against a well-known pool.
func TestAccNVMetNamespace_basic(t *testing.T) {
	t.Skip("nvmet_namespace requires a pre-existing zvol or file; exercise manually or in an integration env")
}
