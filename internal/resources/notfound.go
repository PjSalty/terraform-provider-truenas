package resources

import (
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// isNotFound is the transport-agnostic 404 classifier used by every
// migrated resource's Read path. Each transport's IsNotFound returns
// false for foreign error types, so calling both is safe and the
// resource code stays transport-blind.
//
// Resources that have not yet migrated still call client.IsNotFound
// directly. After Phase 5 (REST client deletion), this helper folds
// down to wsclient.IsNotFound.
func isNotFound(err error) bool {
	return client.IsNotFound(err) || wsclient.IsNotFound(err)
}
