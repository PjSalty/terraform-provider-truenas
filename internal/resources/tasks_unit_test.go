package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --- CronJob ---

func TestCronJobResource_MapResponseToModel_Cases(t *testing.T) {
	r := &CronJobResource{}
	cases := []struct {
		name string
		job  *client.CronJob
	}{
		{name: "basic daily", job: &client.CronJob{ID: 1, User: "root", Command: "/usr/bin/true", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "with stdout", job: &client.CronJob{ID: 2, User: "apps", Command: "/bin/echo", Stdout: true, Stderr: true, Enabled: true, Schedule: client.Schedule{Minute: "*/5", Hour: "*", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "disabled", job: &client.CronJob{ID: 3, User: "u", Command: "/x", Schedule: client.Schedule{Minute: "30", Hour: "1", Dom: "1", Month: "1", Dow: "0"}}},
		{name: "with desc", job: &client.CronJob{ID: 4, User: "root", Command: "backup", Description: "nightly backup", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m CronJobResourceModel
			r.mapResponseToModel(tc.job, &m)
			if m.User.ValueString() != tc.job.User {
				t.Errorf("User mismatch")
			}
			if m.Command.ValueString() != tc.job.Command {
				t.Errorf("Command mismatch")
			}
			if m.Minute.ValueString() != tc.job.Schedule.Minute {
				t.Errorf("Minute mismatch")
			}
			if m.Hour.ValueString() != tc.job.Schedule.Hour {
				t.Errorf("Hour mismatch")
			}
			if m.Enabled.ValueBool() != tc.job.Enabled {
				t.Errorf("Enabled mismatch")
			}
		})
	}
}

func TestCronJobResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewCronJobResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "user", "command", "enabled", "schedule_minute", "schedule_hour"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["command"].IsRequired() {
		t.Error("command should be required")
	}
}

// --- RsyncTask ---

func TestRsyncTaskResource_MapResponseToModel_Cases(t *testing.T) {
	r := &RsyncTaskResource{}
	cases := []struct {
		name string
		task *client.RsyncTask
	}{
		{name: "push ssh", task: &client.RsyncTask{ID: 1, Path: "/mnt/tank/x", Remotehost: "remote", Remoteport: 22, Mode: "SSH", Direction: "PUSH", User: "root", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "4", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "pull module", task: &client.RsyncTask{ID: 2, Path: "/mnt/tank/y", Remotehost: "h2", Mode: "MODULE", Remotemodule: "backup", Direction: "PULL", User: "root", Schedule: client.Schedule{Minute: "15", Hour: "*", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "disabled task", task: &client.RsyncTask{ID: 3, Path: "/x", Remotehost: "h", Mode: "SSH", User: "u", Schedule: client.Schedule{Minute: "0", Hour: "0", Dom: "1", Month: "1", Dow: "1"}}},
		{name: "with desc", task: &client.RsyncTask{ID: 4, Path: "/a", Remotehost: "b", Mode: "SSH", Direction: "PUSH", User: "u", Desc: "hi", Enabled: true, Schedule: client.Schedule{Minute: "*", Hour: "*", Dom: "*", Month: "*", Dow: "*"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m RsyncTaskResourceModel
			r.mapResponseToModel(tc.task, &m)
			if m.Path.ValueString() != tc.task.Path {
				t.Errorf("Path mismatch")
			}
			if m.Remotehost.ValueString() != tc.task.Remotehost {
				t.Errorf("Remotehost mismatch")
			}
			if m.Mode.ValueString() != tc.task.Mode {
				t.Errorf("Mode mismatch")
			}
			if m.Direction.ValueString() != tc.task.Direction {
				t.Errorf("Direction mismatch")
			}
			if m.Enabled.ValueBool() != tc.task.Enabled {
				t.Errorf("Enabled mismatch")
			}
		})
	}
}

func TestRsyncTaskResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewRsyncTaskResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "path", "remotehost", "mode", "direction", "user"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["path"].IsRequired() {
		t.Error("path should be required")
	}
}

// --- ScrubTask ---

func TestScrubTaskResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ScrubTaskResource{}
	cases := []struct {
		name string
		task *client.ScrubTask
	}{
		{name: "basic", task: &client.ScrubTask{ID: 1, Pool: 1, PoolName: "tank", Threshold: 35, Description: "monthly", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "0", Dom: "1", Month: "*", Dow: "*"}}},
		{name: "weekly", task: &client.ScrubTask{ID: 2, Pool: 2, PoolName: "ssd", Threshold: 7, Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "0"}}},
		{name: "disabled", task: &client.ScrubTask{ID: 3, Pool: 3, PoolName: "x", Threshold: 30, Schedule: client.Schedule{Minute: "0", Hour: "1", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "zero threshold", task: &client.ScrubTask{ID: 4, Pool: 4, PoolName: "p4", Threshold: 0, Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "*"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ScrubTaskResourceModel
			r.mapResponseToModel(tc.task, &m)
			if m.Pool.ValueInt64() != int64(tc.task.Pool) {
				t.Errorf("Pool mismatch")
			}
			if m.PoolName.ValueString() != tc.task.PoolName {
				t.Errorf("PoolName mismatch")
			}
			if m.Threshold.ValueInt64() != int64(tc.task.Threshold) {
				t.Errorf("Threshold mismatch")
			}
			if m.Enabled.ValueBool() != tc.task.Enabled {
				t.Errorf("Enabled mismatch")
			}
		})
	}
}

func TestScrubTaskResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewScrubTaskResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "pool", "threshold", "enabled"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
}

// --- SnapshotTask ---

func TestSnapshotTaskResource_MapResponseToModel_Cases(t *testing.T) {
	r := &SnapshotTaskResource{}
	cases := []struct {
		name string
		task *client.SnapshotTask
	}{
		{name: "hourly", task: &client.SnapshotTask{ID: 1, Dataset: "tank/data", Recursive: true, Lifetime: 2, LifetimeUnit: "WEEK", NamingSchema: "auto-%Y-%m-%d_%H-%M", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "*", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "daily", task: &client.SnapshotTask{ID: 2, Dataset: "tank/home", Lifetime: 30, LifetimeUnit: "DAY", NamingSchema: "auto-%Y%m%d", Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*"}}},
		{name: "allow empty", task: &client.SnapshotTask{ID: 3, Dataset: "tank/x", Lifetime: 1, LifetimeUnit: "YEAR", NamingSchema: "auto", AllowEmpty: true, Enabled: true, Schedule: client.Schedule{Minute: "0", Hour: "0", Dom: "1", Month: "1", Dow: "*"}}},
		{name: "disabled", task: &client.SnapshotTask{ID: 4, Dataset: "t/d", Lifetime: 1, LifetimeUnit: "HOUR", NamingSchema: "s", Schedule: client.Schedule{Minute: "0", Hour: "0", Dom: "*", Month: "*", Dow: "*"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m SnapshotTaskResourceModel
			r.mapResponseToModel(tc.task, &m)
			if m.Dataset.ValueString() != tc.task.Dataset {
				t.Errorf("Dataset mismatch")
			}
			if m.Recursive.ValueBool() != tc.task.Recursive {
				t.Errorf("Recursive mismatch")
			}
			if m.Lifetime.ValueInt64() != int64(tc.task.Lifetime) {
				t.Errorf("Lifetime mismatch")
			}
			if m.LifetimeUnit.ValueString() != tc.task.LifetimeUnit {
				t.Errorf("LifetimeUnit mismatch")
			}
			if m.AllowEmpty.ValueBool() != tc.task.AllowEmpty {
				t.Errorf("AllowEmpty mismatch")
			}
		})
	}
}

func TestSnapshotTaskResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewSnapshotTaskResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "dataset", "lifetime_value", "lifetime_unit", "naming_schema"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["dataset"].IsRequired() {
		t.Error("dataset should be required")
	}
}

// --- Replication ---

func TestReplicationResource_MapResponseToModel_Cases(t *testing.T) {
	r := &ReplicationResource{}
	ctx := context.Background()
	cases := []struct {
		name string
		repl *client.Replication
	}{
		{name: "push local", repl: &client.Replication{ID: 1, Name: "r1", Direction: "PUSH", Transport: "LOCAL", SourceDatasets: []string{"tank/a"}, TargetDataset: "tank/b", Enabled: true}},
		{name: "pull ssh", repl: &client.Replication{ID: 2, Name: "r2", Direction: "PULL", Transport: "SSH", SourceDatasets: []string{"remote/a"}, TargetDataset: "tank/r", SSHCredentials: 5, Enabled: true, Recursive: true}},
		{name: "with retention", repl: &client.Replication{ID: 3, Name: "r3", Direction: "PUSH", Transport: "LOCAL", SourceDatasets: []string{"t/s1", "t/s2"}, TargetDataset: "u/d", LifetimeValue: 7, LifetimeUnit: "DAY", RetentionPolicy: "CUSTOM"}},
		{name: "with naming schema", repl: &client.Replication{ID: 4, Name: "r4", Direction: "PUSH", Transport: "SSH", SourceDatasets: []string{"p/s"}, TargetDataset: "p/t", NamingSchema: []string{"auto-%Y-%m-%d"}, AlsoIncludeNamingSchema: []string{"snap-%Y%m%d"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ReplicationResourceModel
			r.mapResponseToModel(ctx, tc.repl, &m)
			if m.Name.ValueString() != tc.repl.Name {
				t.Errorf("Name mismatch")
			}
			if m.Direction.ValueString() != tc.repl.Direction {
				t.Errorf("Direction mismatch")
			}
			if m.Transport.ValueString() != tc.repl.Transport {
				t.Errorf("Transport mismatch")
			}
			if m.TargetDataset.ValueString() != tc.repl.TargetDataset {
				t.Errorf("TargetDataset mismatch")
			}
			if got := len(m.SourceDatasets.Elements()); got != len(tc.repl.SourceDatasets) {
				t.Errorf("SourceDatasets len = %d, want %d", got, len(tc.repl.SourceDatasets))
			}
		})
	}
}

func TestReplicationResource_Schema(t *testing.T) {
	ctx := context.Background()
	r := NewReplicationResource()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", resp.Diagnostics)
	}
	attrs := resp.Schema.GetAttributes()
	for _, k := range []string{"id", "name", "direction", "transport", "source_datasets", "target_dataset"} {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if !attrs["name"].IsRequired() {
		t.Error("name should be required")
	}
}
