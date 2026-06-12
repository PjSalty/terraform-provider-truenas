package datasources

import (
	"context"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestNewCronJobDataSource(t *testing.T) {
	if NewCronJobDataSource() == nil {
		t.Fatal("NewCronJobDataSource returned nil")
	}
}

func TestCronJobDataSource_Schema(t *testing.T) {
	ds := NewCronJobDataSource()
	resp := getDataSourceSchema(t.Context(), t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "user", "command", "description", "enabled",
		"stdout", "stderr", "schedule_minute", "schedule_hour",
		"schedule_dom", "schedule_month", "schedule_dow",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestCronJobDataSource_Read_Success(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.CronJob{
		ID:          5,
		User:        "root",
		Command:     "/usr/bin/backup.sh",
		Description: "nightly backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      false,
		Schedule: truenas.Schedule{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	}))

	ds := NewCronJobDataSource().(*CronJobDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(5)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state CronJobDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.User.ValueString() != "root" {
		t.Errorf("User: got %q", state.User.ValueString())
	}
	if state.Command.ValueString() != "/usr/bin/backup.sh" {
		t.Errorf("Command: got %q", state.Command.ValueString())
	}
	if state.ScheduleMinute.ValueString() != "0" {
		t.Errorf("ScheduleMinute: got %q", state.ScheduleMinute.ValueString())
	}
	if state.ScheduleHour.ValueString() != "2" {
		t.Errorf("ScheduleHour: got %q", state.ScheduleHour.ValueString())
	}
	if !state.Stdout.ValueBool() {
		t.Errorf("Stdout: got %v", state.Stdout.ValueBool())
	}
}

func TestCronJobDataSource_Read_NotFound(t *testing.T) {
	c := newWSServer(t.Context(), t, wsError(wsclient.CodeMethodCallError, "[ENOENT] not found"))

	ds := NewCronJobDataSource().(*CronJobDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(99)})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestCronJobDataSource_Read_DisabledJob(t *testing.T) {
	c := newWSServer(t.Context(), t, wsReturn(truenas.CronJob{
		ID: 1, Enabled: false,
		Schedule: truenas.Schedule{Minute: "*/5"},
	}))

	ds := NewCronJobDataSource().(*CronJobDataSource)
	ds.client = c

	cfg := buildConfig(t.Context(), t, ds, map[string]tftypes.Value{"id": int64Val(1)})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state CronJobDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Enabled.ValueBool() != false {
		t.Errorf("Enabled: got %v", state.Enabled.ValueBool())
	}
	if state.ScheduleMinute.ValueString() != "*/5" {
		t.Errorf("ScheduleMinute: got %q", state.ScheduleMinute.ValueString())
	}
}

func TestCronJobDataSourceAcc(t *testing.T) {
	t.Skip("acceptance test: requires live TrueNAS endpoint")
}
