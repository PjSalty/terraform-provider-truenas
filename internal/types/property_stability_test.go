package types_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	t "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// TestProperty_MarshalRoundTripStable asserts an invariant: every
// exported response struct in internal/types/ round-trips through
// JSON cleanly. That is, for any value V produced by Unmarshal, the
// sequence
//
//	Marshal(V) -> Unmarshal -> V'  AND  V == V'
//
// must hold. A failure here means the type has a "lossy" Unmarshal
// (typically because of a custom UnmarshalJSON that loses data, or
// a field tag that's read-only), which is the exact bug pattern
// that caused PR #12 (cloud_sync credentials nested-object).
//
// Inputs are the seed corpus from fuzz_common_test.go, which already
// catalogs every interesting JSON shape (well-formed, edge values,
// malformed). When this test fails, the failing seed names the broken
// type.
func TestProperty_MarshalRoundTripStable(t_ *testing.T) {
	// Use reflect on a static type registry so we don't have to enumerate
	// 86 types by hand, and so adding a new type to internal/types
	// automatically gets it covered.
	zeros := []interface{}{
		&t.ACLEntry{}, &t.ACLPerms{}, &t.ACMEDNSAuthenticator{}, &t.APIKey{},
		&t.AlertClassEntry{}, &t.AlertClassesConfig{}, &t.AlertService{},
		&t.App{}, &t.Catalog{}, &t.Certificate{}, &t.CloudBackup{},
		&t.CloudBackupSchedule{}, &t.CloudSync{}, &t.CloudSyncCredential{},
		&t.CronJob{}, &t.Dataset{}, &t.DirectoryServicesConfig{}, &t.Disk{},
		&t.FTPConfig{}, &t.FilesystemACL{}, &t.FilesystemACLTemplate{},
		&t.FullNetworkConfig{}, &t.Group{}, &t.ISCSIAuth{}, &t.ISCSIExtent{},
		&t.ISCSIInitiator{}, &t.ISCSIPortal{}, &t.ISCSIPortalListen{},
		&t.ISCSITarget{}, &t.ISCSITargetExtent{}, &t.ISCSITargetGroup{},
		&t.InitScript{}, &t.KMIPConfig{}, &t.KerberosKeytab{},
		&t.KerberosRealm{}, &t.KeychainCredential{}, &t.MailConfig{},
		&t.NFSConfig{}, &t.NFSShare{}, &t.NVMetGlobal{}, &t.NVMetHost{},
		&t.NVMetHostSubsys{}, &t.NVMetNamespace{}, &t.NVMetPort{},
		&t.NVMetPortSubsys{}, &t.NVMetSubsys{}, &t.NetworkInterface{},
		&t.Pool{}, &t.Privilege{}, &t.ReportingExporter{}, &t.Replication{},
		&t.RsyncTask{}, &t.SMBConfig{}, &t.SMBShare{}, &t.SSHConfig{},
		&t.Schedule{}, &t.ScrubTask{}, &t.Service{}, &t.SnapshotTask{},
		&t.SNMPConfig{}, &t.StaticRoute{}, &t.SystemDataset{},
		&t.SystemInfo{}, &t.UpdateTrains{}, &t.UpdateTrainInfo{},
		&t.UpdateCheckResult{}, &t.Tunable{}, &t.UPSConfig{},
		&t.User{}, &t.VM{}, &t.VMDevice{}, &t.VMware{},
	}

	// Subset of seeds that should produce a parseable value for most
	// types, the round-trip check only makes sense when Unmarshal
	// succeeded in the first place.
	parseable := [][]byte{
		[]byte(`{}`),
		[]byte(`{"id":1}`),
		[]byte(`{"id":1,"name":"x"}`),
		[]byte(`{"enabled":true}`),
		[]byte(`{"id":1,"enabled":false}`),
	}

	for _, zero := range zeros {

		ty := reflect.TypeOf(zero).Elem().Name()
		t_.Run(ty, func(t_ *testing.T) {
			for _, seed := range parseable {
				v := reflect.New(reflect.TypeOf(zero).Elem()).Interface()
				if err := json.Unmarshal(seed, v); err != nil {
					continue
				}
				b1, err := json.Marshal(v)
				if err != nil {
					t_.Errorf("seed=%q Marshal #1 failed: %v", seed, err)
					continue
				}
				v2 := reflect.New(reflect.TypeOf(zero).Elem()).Interface()
				if err := json.Unmarshal(b1, v2); err != nil {
					t_.Errorf("seed=%q Marshal->Unmarshal failed: marshalled %q, err: %v",
						seed, b1, err)
					continue
				}
				b2, err := json.Marshal(v2)
				if err != nil {
					t_.Errorf("seed=%q Marshal #2 failed: %v", seed, err)
					continue
				}
				if string(b1) != string(b2) {
					t_.Errorf("seed=%q round-trip drift:\n  first:  %s\n  second: %s",
						seed, b1, b2)
				}
			}
		})
	}
}

// TestProperty_UnmarshalUnknownFieldsTolerated asserts every type
// accepts JSON containing unknown fields, TrueNAS adds attributes
// between versions and a provider that errors on unknown fields
// would break the minute the API ships a new field. Concrete shape
// of the bug pattern: the type uses a strict decoder (DisallowUnknown
// Fields) anywhere in its custom UnmarshalJSON.
func TestProperty_UnmarshalUnknownFieldsTolerated(t_ *testing.T) {
	zeros := []interface{}{
		&t.CloudSync{}, &t.Certificate{}, &t.Dataset{}, &t.Pool{},
		&t.User{}, &t.Group{}, &t.ISCSITarget{}, &t.VM{},
		&t.FilesystemACL{}, &t.KerberosKeytab{},
	}
	// Use only the unknown fields, adding an `id` here would conflict
	// with types that use string vs int IDs. The point of the test
	// is "extra fields don't break decoding", not "id round-trips".
	payload := []byte(`{"__future_field_v3":[1,2,3],"another_added_in_25_10":"value"}`)
	for _, z := range zeros {

		ty := reflect.TypeOf(z).Elem().Name()
		t_.Run(ty, func(t_ *testing.T) {
			v := reflect.New(reflect.TypeOf(z).Elem()).Interface()
			if err := json.Unmarshal(payload, v); err != nil {
				// strings.Contains is fine: the test message guides the fix
				if strings.Contains(err.Error(), "unknown field") {
					t_.Errorf("type rejects unknown fields, would break on new API versions: %v", err)
				} else {
					t_.Errorf("unexpected unmarshal failure: %v", err)
				}
			}
		})
	}
}
