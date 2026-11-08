package provider

import (
	"os"
	"regexp"
	"strings"
	"testing"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// An SSH_CREDENTIALS credential holds its key pair, so the sweeper must
// delete it first or the key pair delete is refused as in use.
func TestKeychainSweepOrder(t *testing.T) {
	in := []truenas.KeychainCredential{
		{ID: 1, Type: "SSH_KEY_PAIR"},
		{ID: 2, Type: "SSH_CREDENTIALS"},
		{ID: 3, Type: "SSH_KEY_PAIR"},
		{ID: 4, Type: "SSH_CREDENTIALS"},
	}
	got := keychainSweepOrder(in)
	want := []int{2, 4, 1, 3}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("order = %v, want ids %v", got, want)
		}
	}
	if in[0].ID != 1 {
		t.Error("keychainSweepOrder reordered its input")
	}
}

// keychainSweeperDepsRE captures the Dependencies list of the keychain
// sweeper's registration. plugin-testing keeps its registry unexported,
// so, like TestSweeperCoverage, this reads sweeper_test.go itself.
var keychainSweeperDepsRE = regexp.MustCompile(
	`(?s)resource\.AddTestSweepers\("truenas_keychain_credential",\s*&resource\.Sweeper\{[^}]*?Dependencies:\s*\[\]string\{([^}]*)\}`)

// A replication task holds its SSH credential, so the keychain sweeper
// must run after the replication sweeper.
func TestKeychainSweeperRunsAfterReplication(t *testing.T) {
	src, err := os.ReadFile("sweeper_test.go")
	if err != nil {
		t.Fatalf("read sweeper_test.go: %v", err)
	}
	m := keychainSweeperDepsRE.FindSubmatch(src)
	if m == nil || !strings.Contains(string(m[1]), `"truenas_replication"`) {
		t.Fatal(`truenas_keychain_credential sweeper must list "truenas_replication" in Dependencies`)
	}
}
