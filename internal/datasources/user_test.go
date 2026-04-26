package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestUserDataSource_Schema(t *testing.T) {
	ds := NewUserDataSource()
	resp := getDataSourceSchema(t, ds)
	attrs := resp.Schema.GetAttributes()
	for _, want := range []string{
		"id", "username", "full_name", "uid", "gid", "home", "shell",
		"locked", "smb", "email", "builtin",
	} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("missing attribute: %s", want)
		}
	}
}

func TestUserDataSource_Read_Success(t *testing.T) {
	email := "alice@example.com"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.User{
			{
				ID:       7,
				UID:      1001,
				Username: "alice",
				FullName: "Alice A",
				Email:    &email,
				Home:     "/home/alice",
				Shell:    "/bin/bash",
				Builtin:  false,
				Locked:   false,
				SMB:      true,
				Group:    client.UserGroup{GID: 1001, Group: "alice"},
			},
		})
	}))

	ds := NewUserDataSource().(*UserDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"username": strVal("alice")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state UserDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.ID.ValueInt64() != 7 {
		t.Errorf("ID: got %d", state.ID.ValueInt64())
	}
	if state.UID.ValueInt64() != 1001 {
		t.Errorf("UID: got %d", state.UID.ValueInt64())
	}
	if state.FullName.ValueString() != "Alice A" {
		t.Errorf("FullName: got %q", state.FullName.ValueString())
	}
	if state.Email.ValueString() != "alice@example.com" {
		t.Errorf("Email: got %q", state.Email.ValueString())
	}
	if state.GID.ValueInt64() != 1001 {
		t.Errorf("GID: got %d", state.GID.ValueInt64())
	}
	if state.SMB.ValueBool() != true {
		t.Errorf("SMB: got %v", state.SMB.ValueBool())
	}
}

func TestUserDataSource_Read_NilEmail(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.User{
			{ID: 1, Username: "bob", Email: nil, Group: client.UserGroup{GID: 100}},
		})
	}))

	ds := NewUserDataSource().(*UserDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"username": strVal("bob")})
	resp := callRead(context.Background(), ds, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var state UserDataSourceModel
	_ = resp.State.Get(context.Background(), &state)
	if state.Email.ValueString() != "" {
		t.Errorf("Email: got %q, want empty", state.Email.ValueString())
	}
}

func TestUserDataSource_Read_NotFound(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []client.User{{Username: "other"}})
	}))

	ds := NewUserDataSource().(*UserDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"username": strVal("missing")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestUserDataSource_Read_ServerError(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
	}))

	ds := NewUserDataSource().(*UserDataSource)
	ds.client = c

	cfg := buildConfig(t, ds, map[string]tftypes.Value{"username": strVal("any")})
	resp := callRead(context.Background(), ds, cfg)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}
