package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// ReplicationEntry returns ssh_credentials as the expanded
// KeychainCredentialEntry (middleware api/v25_10_0/replication.py), not the
// ID that create and update take.
func TestReplication_UnmarshalJSON_SSHCredentialsObject(t *testing.T) {
	body := `{
		"id": 7, "name": "rep", "direction": "PUSH", "transport": "SSH",
		"source_datasets": ["test/src"], "target_dataset": "backup/src",
		"ssh_credentials": {
			"id": 175, "name": "acctest-sshc", "type": "SSH_CREDENTIALS",
			"attributes": {"host": "192.0.2.10", "port": 22, "private_key": 174}
		}
	}`
	var r Replication
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.SSHCredentials != 175 {
		t.Errorf("SSHCredentials = %d, want 175", r.SSHCredentials)
	}
	if r.ID != 7 || r.Name != "rep" || r.Transport != "SSH" || r.TargetDataset != "backup/src" {
		t.Errorf("other fields lost: %+v", r)
	}
}

func TestReplication_UnmarshalJSON_SSHCredentialsShapes(t *testing.T) {
	cases := map[string]struct {
		field string
		want  int
	}{
		"plain id": {`"ssh_credentials": 5,`, 5},
		"null":     {`"ssh_credentials": null,`, 0},
		"absent":   {``, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var r Replication
			if err := json.Unmarshal([]byte(`{`+tc.field+`"id": 1}`), &r); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.SSHCredentials != tc.want || r.ID != 1 {
				t.Errorf("got SSHCredentials=%d ID=%d, want %d and 1", r.SSHCredentials, r.ID, tc.want)
			}
		})
	}
}

func TestReplication_UnmarshalJSON_Errors(t *testing.T) {
	var r Replication
	if err := json.Unmarshal([]byte(`{"ssh_credentials": "x"}`), &r); err == nil ||
		!strings.Contains(err.Error(), "ssh_credentials") {
		t.Errorf("a string ssh_credentials gave %v, want an error naming the field", err)
	}
	if err := json.Unmarshal([]byte(`{"id": "not a number"}`), &r); err == nil {
		t.Error("a malformed body was accepted")
	}
	if err := json.Unmarshal([]byte(`{"periodic_snapshot_tasks": [3, "x"]}`), &r); err == nil ||
		!strings.Contains(err.Error(), "periodic_snapshot_tasks") {
		t.Errorf("a string snapshot task gave %v, want an error naming the field", err)
	}
}

// ReplicationEntry returns periodic_snapshot_tasks as the expanded
// PoolSnapshotTaskDBEntry list (middleware api/v25_10_0/replication.py), not
// the IDs that create and update take.
func TestReplication_UnmarshalJSON_PeriodicSnapshotTasks(t *testing.T) {
	cases := map[string]struct {
		field string
		want  []int
	}{
		"expanded entries": {`"periodic_snapshot_tasks": [{"id": 4, "dataset": "tank/a"}, {"id": 9, "dataset": "tank/b"}],`, []int{4, 9}},
		"plain ids":        {`"periodic_snapshot_tasks": [4, 9],`, []int{4, 9}},
		"empty":            {`"periodic_snapshot_tasks": [],`, []int{}},
		"null":             {`"periodic_snapshot_tasks": null,`, []int{}},
		"absent":           {``, []int{}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var r Replication
			if err := json.Unmarshal([]byte(`{`+tc.field+`"id": 1}`), &r); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.PeriodicSnapshotTasks == nil {
				t.Fatal("PeriodicSnapshotTasks is nil, want a list even when there are no tasks")
			}
			if len(r.PeriodicSnapshotTasks) != len(tc.want) {
				t.Fatalf("PeriodicSnapshotTasks = %v, want %v", r.PeriodicSnapshotTasks, tc.want)
			}
			for i, id := range tc.want {
				if r.PeriodicSnapshotTasks[i] != id {
					t.Errorf("PeriodicSnapshotTasks = %v, want %v", r.PeriodicSnapshotTasks, tc.want)
				}
			}
		})
	}
}

func TestReplication_UnmarshalJSON_Schedules(t *testing.T) {
	body := `{
		"id": 3,
		"schedule": {"minute": "00", "hour": "*/6", "dom": "*", "month": "*", "dow": "*", "begin": "00:00", "end": "23:59"},
		"restrict_schedule": null,
		"only_matching_schedule": true,
		"hold_pending_snapshots": true
	}`
	var r Replication
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := TimeSchedule{Minute: "00", Hour: "*/6", Dom: "*", Month: "*", Dow: "*", Begin: "00:00", End: "23:59"}
	if r.Schedule == nil || *r.Schedule != want {
		t.Errorf("Schedule = %+v, want %+v", r.Schedule, want)
	}
	if r.RestrictSchedule != nil {
		t.Errorf("RestrictSchedule = %+v, want nil for a null", r.RestrictSchedule)
	}
	if !r.OnlyMatchingSchedule || !r.HoldPendingSnapshots {
		t.Errorf("flags lost: only_matching_schedule=%v hold_pending_snapshots=%v", r.OnlyMatchingSchedule, r.HoldPendingSnapshots)
	}
}

// A schedule field has three request forms, and TrueNAS reads each one
// differently: a missing key keeps the stored schedule, null removes it.
func TestNullable_JSON(t *testing.T) {
	cases := map[string]struct {
		field Nullable[TimeSchedule]
		want  string
	}{
		"zero value leaves the key out": {Nullable[TimeSchedule]{}, `{"name":"r"}`},
		"clear sends null":              {Null[TimeSchedule](), `{"name":"r","schedule":null}`},
		"value sends only the set fields": {
			Value(TimeSchedule{Hour: "*/6", End: "18:00"}),
			`{"name":"r","schedule":{"hour":"*/6","end":"18:00"}}`,
		},
		"an entry with no fields is an all-default schedule": {Value(TimeSchedule{}), `{"name":"r","schedule":{}}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := json.Marshal(ReplicationUpdateRequest{Name: "r", Schedule: tc.field})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("request = %s, want %s", got, tc.want)
			}
		})
	}
}

// An empty list has to reach TrueNAS as [], because that is what clears a
// list field. A nil pointer leaves the key out.
func TestReplicationUpdateRequest_EmptyListsAreSent(t *testing.T) {
	got, err := json.Marshal(ReplicationUpdateRequest{
		PeriodicSnapshotTasks:   &[]int{},
		NamingSchema:            &[]string{},
		AlsoIncludeNamingSchema: &[]string{"auto-%Y-%m-%d_%H-%M"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"periodic_snapshot_tasks":[],"naming_schema":[],"also_include_naming_schema":["auto-%Y-%m-%d_%H-%M"]}`
	if string(got) != want {
		t.Errorf("request = %s, want %s", got, want)
	}
	if got, _ := json.Marshal(ReplicationUpdateRequest{}); string(got) != `{}` {
		t.Errorf("an empty update request = %s, want {}", got)
	}
}

func TestReplicationCreateRequest_OptionalFieldsLeftOut(t *testing.T) {
	got, err := json.Marshal(ReplicationCreateRequest{Name: "r", Direction: "PUSH", Transport: "LOCAL", TargetDataset: "t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, key := range []string{"schedule", "restrict_schedule", "only_matching_schedule", "hold_pending_snapshots", "periodic_snapshot_tasks"} {
		if strings.Contains(string(got), `"`+key+`"`) {
			t.Errorf("%s was sent although it was not set: %s", key, got)
		}
	}
	held := true
	got, err = json.Marshal(ReplicationCreateRequest{
		Name: "r", HoldPendingSnapshots: &held, PeriodicSnapshotTasks: &[]int{4},
		RestrictSchedule: Value(TimeSchedule{Hour: "*/2"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, part := range []string{`"hold_pending_snapshots":true`, `"periodic_snapshot_tasks":[4]`, `"restrict_schedule":{"hour":"*/2"}`} {
		if !strings.Contains(string(got), part) {
			t.Errorf("request lacks %s: %s", part, got)
		}
	}
}

// The options a create and an update share: nothing is sent until it is set,
// and the empty forms that clear a field are sent as written.
func TestReplicationOptions_JSON(t *testing.T) {
	got, err := json.Marshal(ReplicationUpdateRequest{})
	if err != nil || string(got) != `{}` {
		t.Fatalf("empty options = %s (%v), want {}", got, err)
	}

	yes, retries := true, 3
	set := ReplicationOptions{
		Readonly: "IGNORE", AllowFromScratch: &yes, Retries: &retries,
		Exclude: &[]string{"tank/a/tmp"}, PropertiesOverride: map[string]string{"atime": "off"},
		Compression: Value("LZ4"), SpeedLimit: Value(1048576), NameRegex: Value("auto-.*"),
		Lifetimes: &[]ReplicationLifetime{{
			Schedule:      Schedule{Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "7"},
			LifetimeValue: 4, LifetimeUnit: "WEEK",
		}},
	}
	got, err = json.Marshal(ReplicationCreateRequest{Name: "r", ReplicationOptions: set})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, part := range []string{
		`"readonly":"IGNORE"`, `"allow_from_scratch":true`, `"retries":3`, `"exclude":["tank/a/tmp"]`,
		`"properties_override":{"atime":"off"}`, `"compression":"LZ4"`, `"speed_limit":1048576`, `"name_regex":"auto-.*"`,
		`"lifetimes":[{"schedule":{"minute":"0","hour":"0","dom":"*","month":"*","dow":"7"},"lifetime_value":4,"lifetime_unit":"WEEK"}]`,
	} {
		if !strings.Contains(string(got), part) {
			t.Errorf("request lacks %s: %s", part, got)
		}
	}

	cleared := ReplicationOptions{
		Exclude: &[]string{}, PropertiesExclude: &[]string{}, PropertiesOverride: map[string]string{},
		Compression: Null[string](), SpeedLimit: Null[int](), LoggingLevel: Null[string](), NameRegex: Null[string](),
		Lifetimes: &[]ReplicationLifetime{},
	}
	got, err = json.Marshal(cleared)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"exclude":[],"properties_exclude":[],"properties_override":{},"compression":null,"speed_limit":null,` +
		`"logging_level":null,"name_regex":null,"lifetimes":[]}`
	if string(got) != want {
		t.Errorf("cleared options = %s, want %s", got, want)
	}
}

func TestReplication_UnmarshalJSON_Options(t *testing.T) {
	body := `{
		"id": 3, "readonly": "REQUIRE", "allow_from_scratch": true, "retries": 7,
		"exclude": ["tank/a/tmp"], "properties": false, "properties_exclude": ["mountpoint"],
		"properties_override": {"atime": "off"}, "replicate": true,
		"compression": "PIGZ", "speed_limit": 2048, "large_block": false, "embed": true, "compressed": false,
		"sudo": true, "logging_level": null, "name_regex": null,
		"lifetimes": [{"schedule": {"minute": "0", "hour": "0", "dom": "1", "month": "*", "dow": "*"},
			"lifetime_value": 12, "lifetime_unit": "MONTH"}]
	}`
	var r Replication
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Readonly != "REQUIRE" || !r.AllowFromScratch || r.Retries != 7 || r.Properties || !r.Replicate ||
		r.LargeBlock || !r.Embed || r.Compressed || !r.Sudo {
		t.Errorf("scalar options lost: %+v", r)
	}
	if r.Compression == nil || *r.Compression != "PIGZ" || r.SpeedLimit == nil || *r.SpeedLimit != 2048 {
		t.Errorf("compression = %v, speed_limit = %v, want PIGZ and 2048", r.Compression, r.SpeedLimit)
	}
	if r.LoggingLevel != nil || r.NameRegex != nil {
		t.Errorf("logging_level = %v, name_regex = %v, want nil for null", r.LoggingLevel, r.NameRegex)
	}
	if len(r.Exclude) != 1 || len(r.PropertiesExclude) != 1 || r.PropertiesOverride["atime"] != "off" {
		t.Errorf("collections lost: %v %v %v", r.Exclude, r.PropertiesExclude, r.PropertiesOverride)
	}
	if len(r.Lifetimes) != 1 || r.Lifetimes[0].Schedule.Dom != "1" || r.Lifetimes[0].LifetimeValue != 12 ||
		r.Lifetimes[0].LifetimeUnit != "MONTH" {
		t.Errorf("lifetimes = %+v", r.Lifetimes)
	}
}

func TestReplicationOptions_EncryptionAndNetcatJSON(t *testing.T) {
	on := true
	got, err := json.Marshal(ReplicationOptions{
		Encryption: &on, EncryptionKey: Value("00ff"), EncryptionKeyFormat: Value("HEX"),
		NetcatActiveSide: Value("LOCAL"), NetcatActiveSidePortMin: Value(2000), NetcatActiveSidePortMax: Null[int](),
		EncryptionKeyLocation: Null[string](),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"encryption":true,"encryption_key":"00ff","encryption_key_format":"HEX","encryption_key_location":null,` +
		`"netcat_active_side":"LOCAL","netcat_active_side_port_min":2000,"netcat_active_side_port_max":null}`
	if string(got) != want {
		t.Errorf("options = %s, want %s", got, want)
	}

	var r Replication
	body := `{"id": 1, "encryption": true, "encryption_inherit": null, "encryption_key": "00ff",
		"encryption_key_format": "HEX", "encryption_key_location": null, "netcat_active_side": "REMOTE",
		"netcat_active_side_listen_address": "192.0.2.5", "netcat_active_side_port_min": 2000,
		"netcat_active_side_port_max": 2100, "netcat_passive_side_connect_address": null}`
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Encryption || r.EncryptionInherit != nil || r.EncryptionKey == nil || *r.EncryptionKey != "00ff" ||
		r.EncryptionKeyLocation != nil || r.NetcatActiveSide == nil || *r.NetcatActiveSide != "REMOTE" ||
		r.NetcatActiveSidePortMin == nil || *r.NetcatActiveSidePortMin != 2000 || r.NetcatPassiveSideConnectAddress != nil {
		t.Errorf("decoded entry lost values: %+v", r)
	}
}
