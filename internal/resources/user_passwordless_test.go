package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// callValidateConfig drives ValidateConfig with a config built from the
// given attribute map.
func callValidateConfig(t *testing.T, r *UserResource, vals map[string]tftypes.Value) *resource.ValidateConfigResponse {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	raw := rawFromValues(t, ctx, sch, vals)
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: sch.Schema, Raw: raw},
	}, resp)
	return resp
}

// Mirrors middleware: 'Leave "Password" blank when "Disable password login"
// is checked.' Catching it here gives an attribute path instead of an opaque
// server-side validation error.
func TestUserResource_ValidateConfig_passwordWithDisabledIsAnError(t *testing.T) {
	r := &UserResource{}
	resp := callValidateConfig(t, r, map[string]tftypes.Value{
		"username":          tftypes.NewValue(tftypes.String, "svc"),
		"password":          tftypes.NewValue(tftypes.String, "hunter2"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("password alongside password_disabled was accepted; middleware rejects it")
	}
}

// Mirrors middleware: 'Password authentication may not be disabled for SMB
// users.' Samba needs a password to derive the NT hash.
func TestUserResource_ValidateConfig_smbWithDisabledIsAnError(t *testing.T) {
	r := &UserResource{}
	resp := callValidateConfig(t, r, map[string]tftypes.Value{
		"username":          tftypes.NewValue(tftypes.String, "svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		"smb":               tftypes.NewValue(tftypes.Bool, true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("an SMB user with password_disabled was accepted; SMB auth would silently never work")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "SMB") {
		t.Errorf("diagnostic should explain the SMB conflict: %v", resp.Diagnostics)
	}
}

// smb left unset takes the schema default of false, so there is no conflict.
func TestUserResource_ValidateConfig_disabledWithoutSMBIsFine(t *testing.T) {
	r := &UserResource{}
	resp := callValidateConfig(t, r, map[string]tftypes.Value{
		"username":          tftypes.NewValue(tftypes.String, "svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	if resp.Diagnostics.HasError() {
		t.Errorf("a plain passwordless account was rejected: %v", resp.Diagnostics)
	}
}

// A password with password_disabled unset or false is the ordinary case.
func TestUserResource_ValidateConfig_passwordAloneIsFine(t *testing.T) {
	r := &UserResource{}
	resp := callValidateConfig(t, r, map[string]tftypes.Value{
		"username": tftypes.NewValue(tftypes.String, "svc"),
		"password": tftypes.NewValue(tftypes.String, "hunter2"),
	})
	if resp.Diagnostics.HasError() {
		t.Errorf("an ordinary password-bearing user was rejected: %v", resp.Diagnostics)
	}
}

// --- ModifyPlan ---

// callUserModifyPlan drives ModifyPlan with explicit plan and state raws so
// the create-vs-update branch can be selected.
func callUserModifyPlan(t *testing.T, r *UserResource, planVals map[string]tftypes.Value, stateVals map[string]tftypes.Value) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	planRaw := rawFromValues(t, ctx, sch, planVals)

	stateRaw := tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)
	if stateVals != nil {
		stateRaw = rawFromValues(t, ctx, sch, stateVals)
	}
	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: sch.Schema, Raw: planRaw},
		Plan:   tfsdk.Plan{Schema: sch.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: sch.Schema, Raw: stateRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: planRaw}}
	r.ModifyPlan(ctx, req, resp)
	return resp
}

// Create with neither a password nor password_disabled is what middleware
// answers with "Password is required". Fail at plan time instead.
func TestUserResource_ModifyPlan_createWithoutPasswordOrDisabled(t *testing.T) {
	r := &UserResource{}
	resp := callUserModifyPlan(t, r, map[string]tftypes.Value{
		"username": tftypes.NewValue(tftypes.String, "svc"),
	}, nil)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a create with no password and no password_disabled was allowed")
	}
}

func TestUserResource_ModifyPlan_createPasswordlessIsAllowed(t *testing.T) {
	r := &UserResource{}
	resp := callUserModifyPlan(t, r, map[string]tftypes.Value{
		"username":          tftypes.NewValue(tftypes.String, "svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	}, nil)
	if resp.Diagnostics.HasError() {
		t.Errorf("a deliberate passwordless create was blocked: %v", resp.Diagnostics)
	}
}

func TestUserResource_ModifyPlan_createWithPasswordIsAllowed(t *testing.T) {
	r := &UserResource{}
	resp := callUserModifyPlan(t, r, map[string]tftypes.Value{
		"username": tftypes.NewValue(tftypes.String, "svc"),
		"password": tftypes.NewValue(tftypes.String, "hunter2"),
	}, nil)
	if resp.Diagnostics.HasError() {
		t.Errorf("an ordinary create was blocked: %v", resp.Diagnostics)
	}
}

// Turning password_disabled back off without supplying a password would
// leave an account claiming password login whose hash is still '*'. This is
// also the guard that catches an imported passwordless account whose config
// simply forgot password_disabled, which the schema Default(false) would
// otherwise flip silently.
func TestUserResource_ModifyPlan_reenablingWithoutPasswordIsAnError(t *testing.T) {
	r := &UserResource{}
	resp := callUserModifyPlan(t,
		r,
		map[string]tftypes.Value{
			"username":          tftypes.NewValue(tftypes.String, "svc"),
			"password_disabled": tftypes.NewValue(tftypes.Bool, false),
		},
		map[string]tftypes.Value{
			"username":          tftypes.NewValue(tftypes.String, "svc"),
			"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		})
	if !resp.Diagnostics.HasError() {
		t.Fatal("password auth was re-enabled with no password; the account would have hash '*'")
	}
}

func TestUserResource_ModifyPlan_reenablingWithPasswordIsAllowed(t *testing.T) {
	r := &UserResource{}
	resp := callUserModifyPlan(t,
		r,
		map[string]tftypes.Value{
			"username":          tftypes.NewValue(tftypes.String, "svc"),
			"password":          tftypes.NewValue(tftypes.String, "hunter2"),
			"password_disabled": tftypes.NewValue(tftypes.Bool, false),
		},
		map[string]tftypes.Value{
			"username":          tftypes.NewValue(tftypes.String, "svc"),
			"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		})
	if resp.Diagnostics.HasError() {
		t.Errorf("re-enabling with a password was blocked: %v", resp.Diagnostics)
	}
}

// Staying passwordless across an update must not demand a password.
func TestUserResource_ModifyPlan_stayingDisabledIsAllowed(t *testing.T) {
	r := &UserResource{}
	resp := callUserModifyPlan(t,
		r,
		map[string]tftypes.Value{
			"username":          tftypes.NewValue(tftypes.String, "svc"),
			"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		},
		map[string]tftypes.Value{
			"username":          tftypes.NewValue(tftypes.String, "svc"),
			"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		})
	if resp.Diagnostics.HasError() {
		t.Errorf("an unchanged passwordless account was blocked: %v", resp.Diagnostics)
	}
}

func TestUserResource_ModifyPlan_destroyIsANoop(t *testing.T) {
	ctx := context.Background()
	r := &UserResource{}
	sch := schemaOf(t, ctx, r)
	nullRaw := tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)
	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: sch.Schema, Raw: nullRaw},
		Plan:   tfsdk.Plan{Schema: sch.Schema, Raw: nullRaw},
		State:  tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)},
	}
	resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: nullRaw}}
	r.ModifyPlan(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("destroy raised an error: %v", resp.Diagnostics)
	}
}

// --- wire-level: what Update actually sends ---

// userUpdateRecorder captures the user.update params so the password key's
// three states can be told apart.
func userUpdateRecorder(ctx context.Context, t *testing.T, entity map[string]interface{}, got *map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "user.update":
			if len(params) > 1 {
				if m, ok := params[1].(map[string]interface{}); ok {
					*got = m
				}
			}
			return entity, nil
		case "user.get_instance", "user.query":
			if method == "user.query" {
				return []interface{}{entity}, nil
			}
			return entity, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func passwordlessUserEntity(disabled bool) map[string]interface{} {
	return map[string]interface{}{
		"id": 1, "uid": 1000, "username": "svc", "full_name": "svc",
		"email": nil, "home": "/var/empty", "shell": "/usr/sbin/nologin",
		"locked": false, "smb": false, "password_disabled": disabled,
		"group":  map[string]interface{}{"id": 100, "bsdgrp_gid": 100},
		"groups": []int{}, "sudo_commands": []string{}, "sshpubkey": nil,
	}
}

// Disabling password login must put an EXPLICIT null on the wire. Omitting
// the key would leave the stored hash intact, so the account would still
// accept its old password while Terraform reported it as passwordless.
func TestUserResource_Update_disablingSendsExplicitNull(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, passwordlessUserEntity(true), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, false),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}

	v, present := got["password"]
	if !present {
		t.Fatalf("password key absent; the stored hash would survive: %v", got)
	}
	if v != nil {
		t.Errorf("password = %v, want an explicit null", v)
	}
	if got["password_disabled"] != true {
		t.Errorf("password_disabled = %v, want true", got["password_disabled"])
	}
}

// An update that neither sets nor disables a password must omit the key, so
// an account can be managed without rotating its password.
func TestUserResource_Update_noPasswordChangeOmitsTheKey(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, passwordlessUserEntity(false), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("old"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, false),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("new"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, false),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	if _, present := got["password"]; present {
		t.Errorf("password key sent on an update that did not touch it: %v", got)
	}
}

// password_disabled has to round-trip from the wire into state, or drift is
// invisible and the ModifyPlan re-enable guard never fires.
func TestUserResource_Read_populatesPasswordDisabled(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, passwordlessUserEntity(true), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", rResp.Diagnostics)
	}
	var model UserResourceModel
	rResp.State.Get(ctx, &model)
	if !model.PasswordDisabled.ValueBool() {
		t.Error("password_disabled was not read back from the wire")
	}
}

// userCreateRecorder captures the user.create body.
func userCreateRecorder(ctx context.Context, t *testing.T, entity map[string]interface{}, got *map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "user.create":
			if len(params) > 0 {
				if m, ok := params[0].(map[string]interface{}); ok {
					*got = m
				}
			}
			return entity, nil
		case "user.get_instance":
			return entity, nil
		case "user.query":
			return []interface{}{entity}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func TestUserResource_Create_sendsPasswordWhenSet(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userCreateRecorder(ctx, t, passwordlessUserEntity(false), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"username": str("svc"), "full_name": str("svc"),
		"password":          str("hunter2"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, false),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
	if got["password"] != "hunter2" {
		t.Errorf("password = %v, want hunter2; the account would be created without one", got["password"])
	}
}

// The passwordless create must omit the key entirely. "" is rejected by
// middleware's NonEmptyString model, and sending it alongside
// password_disabled is refused outright.
func TestUserResource_Create_passwordlessOmitsTheKey(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userCreateRecorder(ctx, t, passwordlessUserEntity(true), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"username": str("svc"), "full_name": str("svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
	if _, present := got["password"]; present {
		t.Errorf("password key sent on a passwordless create: %v", got)
	}
	if got["password_disabled"] != true {
		t.Errorf("password_disabled = %v, want true", got["password_disabled"])
	}
}
