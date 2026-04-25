package main

import (
	"context"
	"errors"
	"os"
	"testing"

	pfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// TestMain_Dispatches verifies main() → run() → serveFn with a stubbed
// serve function so we don't actually bind a provider socket. os.Args is
// reset so that the flag parser in run() doesn't trip on go test's own
// -test.* flags.
func TestMain_Dispatches(t *testing.T) {
	origServe := serveFn
	origFatal := logFatal
	origArgs := os.Args
	t.Cleanup(func() { serveFn = origServe; logFatal = origFatal; os.Args = origArgs })

	os.Args = []string{"terraform-provider-truenas"}
	var called bool
	serveFn = func(_ context.Context, factory func() pfprovider.Provider, _ providerserver.ServeOpts) error {
		// Invoke the factory so the inner closure line is covered.
		_ = factory()
		called = true
		return nil
	}
	logFatal = func(_ ...any) { /* swallow */ }

	main()

	if !called {
		t.Fatal("expected serveFn to be called by main()")
	}
}

// TestRun_Debug ensures the -debug flag is parsed and passed through.
func TestRun_Debug(t *testing.T) {
	origServe := serveFn
	origFatal := logFatal
	t.Cleanup(func() { serveFn = origServe; logFatal = origFatal })

	var gotDebug bool
	serveFn = func(_ context.Context, _ func() pfprovider.Provider, opts providerserver.ServeOpts) error {
		gotDebug = opts.Debug
		return nil
	}
	logFatal = func(_ ...any) {}

	run([]string{"-debug"})

	if !gotDebug {
		t.Error("expected ServeOpts.Debug=true when -debug flag is set")
	}
}

// TestRun_ServeError verifies that a serveFn error is routed to logFatal.
func TestRun_ServeError(t *testing.T) {
	origServe := serveFn
	origFatal := logFatal
	t.Cleanup(func() { serveFn = origServe; logFatal = origFatal })

	serveFn = func(_ context.Context, _ func() pfprovider.Provider, _ providerserver.ServeOpts) error {
		return errors.New("boom")
	}
	var fatalCalled bool
	logFatal = func(_ ...any) { fatalCalled = true }

	run(nil)

	if !fatalCalled {
		t.Error("expected logFatal to be called on serve error")
	}
}

// TestRun_BadFlag exercises the flag-parse error branch.
func TestRun_BadFlag(t *testing.T) {
	origServe := serveFn
	origFatal := logFatal
	t.Cleanup(func() { serveFn = origServe; logFatal = origFatal })

	serveFn = func(_ context.Context, _ func() pfprovider.Provider, _ providerserver.ServeOpts) error {
		t.Fatal("serveFn should not be called when flag parse fails")
		return nil
	}
	var fatalCalled bool
	logFatal = func(_ ...any) { fatalCalled = true }

	run([]string{"-nonexistent-flag"})

	if !fatalCalled {
		t.Error("expected logFatal to be called on bad flag")
	}
}
