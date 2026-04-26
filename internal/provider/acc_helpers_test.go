package provider

import (
	"context"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// These package-local helpers used to own the full implementation of the
// shared acceptance-test boilerplate. They now delegate to
// internal/acctest so a single copy of the logic lives in one place while
// the ~60 existing acc_*_test.go call sites keep compiling without a
// churn-heavy rename pass. New tests SHOULD call internal/acctest
// directly; the wrappers exist only to avoid rewriting 60 files.

// testAccClient builds a live client.Client from the same environment
// variables the provider uses. See internal/acctest.Client.
func testAccClient() (*client.Client, error) {
	return acctest.Client()
}

// testAccCtx returns a short-lived context suitable for out-of-band API
// calls made from within `_disappears` test check functions.
// See internal/acctest.Ctx.
func testAccCtx() (context.Context, context.CancelFunc) {
	return acctest.Ctx()
}

// randomName returns a unique resource name with the given prefix so
// that concurrent or back-to-back acceptance test runs never collide on
// the TrueNAS side. See internal/acctest.RandomName.
func randomName(prefix string) string {
	return acctest.RandomName(prefix)
}

// shortSuffix returns a short numeric suffix (last 6 digits of UnixNano)
// for resources with tight length limits. See internal/acctest.ShortSuffix.
func shortSuffix() int64 {
	return acctest.ShortSuffix()
}

// pidSuffix is an alias of shortSuffix used by tests that need a short
// numeric token for fields with length constraints.
// See internal/acctest.PIDSuffix.
func pidSuffix() int64 {
	return acctest.PIDSuffix()
}

// skipMsg is the shared message used by every acceptance test when
// TF_ACC is not set. See internal/acctest.SkipMsg.
const skipMsg = acctest.SkipMsg
