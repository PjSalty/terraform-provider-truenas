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
func TestScheduleRequest_JSON(t *testing.T) {
	cases := map[string]struct {
		field ScheduleRequest
		want  string
	}{
		"zero value leaves the key out": {ScheduleRequest{}, `{"name":"r"}`},
		"clear sends null":              {ScheduleRequest{Clear: true}, `{"name":"r","schedule":null}`},
		"value sends only the set fields": {
			ScheduleRequest{Value: &TimeSchedule{Hour: "*/6", End: "18:00"}},
			`{"name":"r","schedule":{"hour":"*/6","end":"18:00"}}`,
		},
		"an entry with no fields is an all-default schedule": {ScheduleRequest{Value: &TimeSchedule{}}, `{"name":"r","schedule":{}}`},
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
		RestrictSchedule: ScheduleRequest{Value: &TimeSchedule{Hour: "*/2"}},
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
