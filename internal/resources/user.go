package resources

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                   = &UserResource{}
	_ resource.ResourceWithImportState    = &UserResource{}
	_ resource.ResourceWithModifyPlan     = &UserResource{}
	_ resource.ResourceWithValidateConfig = &UserResource{}
)

// UserResource manages a TrueNAS local user.
type UserResource struct {
	client *wsclient.Client
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	UID              types.Int64    `tfsdk:"uid"`
	Username         types.String   `tfsdk:"username"`
	FullName         types.String   `tfsdk:"full_name"`
	Email            types.String   `tfsdk:"email"`
	Password         types.String   `tfsdk:"password"`
	PasswordDisabled types.Bool     `tfsdk:"password_disabled"`
	Webshare         types.Bool     `tfsdk:"webshare"`
	Group            types.Int64    `tfsdk:"group"`
	GroupCreate      types.Bool     `tfsdk:"group_create"`
	Groups           types.Set      `tfsdk:"groups"`
	Home             types.String   `tfsdk:"home"`
	Shell            types.String   `tfsdk:"shell"`
	Locked           types.Bool     `tfsdk:"locked"`
	SMB              types.Bool     `tfsdk:"smb"`
	SSHPubKey        types.String   `tfsdk:"sshpubkey"`
	SudoCommands     types.List     `tfsdk:"sudo_commands"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a local user on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.Int64Attribute{
				Description: "The UNIX UID for the user. If not set, TrueNAS will assign one.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 4294967295),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The login name of the user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z_][a-z0-9_-]*\$?$`),
						"must be a valid POSIX username (lowercase letters, digits, underscore, dash; must start with letter or underscore)",
					),
				},
			},
			"full_name": schema.StringAttribute{
				Description: "The full (display) name of the user.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"email": schema.StringAttribute{
				Description: "Email address of the user.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
			},
			// Optional, not Required: a service account that only owns
			// files (an NFS mapall user, for instance) is legitimately
			// passwordless, and TrueNAS models that with
			// password_disabled. LengthAtLeast still rejects an explicit
			// "", because framework-validators short-circuits on null.
			"password": schema.StringAttribute{
				Description: "The password for the user. Omit it and set `password_disabled = true` " +
					"for an account that should have no password login.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			// New in TrueNAS 26.0. Setting it makes middleware add the
			// account to the builtin truenas_webshare group.
			//
			// Modeling it is not optional: middleware's handle_webshare
			// runs on every user.update and, when the payload omits
			// webshare, inherits the PRE-update value and forces group
			// membership to match. An unmodeled field therefore silently
			// re-added the group after every apply, so Terraform saw a
			// group set it did not ask for and failed with "Provider
			// produced inconsistent result after apply" on every run,
			// unrepairable through configuration.
			"webshare": schema.BoolAttribute{
				Description: "Allow this account to access WebShare shares. Requires TrueNAS 26.0 or newer. " +
					"Setting it adds the user to the builtin `truenas_webshare` group.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"password_disabled": schema.BoolAttribute{
				Description: "Disable password authentication for this user. Cannot be combined with " +
					"`password`, and TrueNAS rejects it for SMB users.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"group": schema.Int64Attribute{
				Description: "The primary group ID. If not specified and group_create is true, a group matching the username will be created.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 4294967295),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"group_create": schema.BoolAttribute{
				Description: "Whether to create a new primary group for the user.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"groups": schema.SetAttribute{
				Description: "Auxiliary group IDs. This is an unordered set; the order of IDs is not significant.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Default:     setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{})),
			},
			"home": schema.StringAttribute{
				Description: "Home directory path. Must begin with /mnt or be /var/empty.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/var/empty"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(/mnt/|/var/empty$|/nonexistent$|/root$|/home/)`),
						"home directory must start with /mnt/, be /var/empty, /nonexistent, /root, or start with /home/",
					),
				},
			},
			"shell": schema.StringAttribute{
				Description: "Login shell (e.g., /usr/sbin/nologin, /bin/bash).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/usr/sbin/nologin"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the user account is locked.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the user should have Samba authentication enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sshpubkey": schema.StringAttribute{
				Description: "SSH public key for the user.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"sudo_commands": schema.ListAttribute{
				Description: "List of sudo commands the user is allowed to run.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*wsclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *wsclient.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create User start")

	var plan UserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	passwordDisabled := plan.PasswordDisabled.ValueBool()
	createReq := &truenas.UserCreateRequest{
		Username:         plan.Username.ValueString(),
		FullName:         plan.FullName.ValueString(),
		GroupCreate:      plan.GroupCreate.ValueBool(),
		Locked:           plan.Locked.ValueBool(),
		SMB:              plan.SMB.ValueBool(),
		PasswordDisabled: &passwordDisabled,
	}
	// Only sent when the server has the field. The 26.0 models are
	// extra="forbid", and so are the 25.10 ones, so sending webshare to a
	// 25.10 box is a hard ValidationError rather than an ignored key.
	if err := setUserWebshare(ctx, r.client, plan.Webshare, func(b *bool) { createReq.Webshare = b }); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("webshare"), "Invalid Webshare", err.Error())
		return
	}
	// The key is omitted entirely when no password was given. Sending ""
	// would be rejected by the NonEmptyString model, and sending it
	// alongside password_disabled is refused outright by middleware.
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		pw := plan.Password.ValueString()
		createReq.Password = &pw
	}

	if !plan.Email.IsNull() && plan.Email.ValueString() != "" {
		createReq.Email = plan.Email.ValueString()
	}
	if !plan.UID.IsNull() && !plan.UID.IsUnknown() {
		createReq.UID = int(plan.UID.ValueInt64())
	}
	if !plan.Group.IsNull() && !plan.Group.IsUnknown() {
		createReq.Group = int(plan.Group.ValueInt64())
		createReq.GroupCreate = false
	}
	if !plan.Home.IsNull() {
		createReq.Home = plan.Home.ValueString()
	}
	if !plan.Shell.IsNull() {
		createReq.Shell = plan.Shell.ValueString()
	}
	if !plan.SSHPubKey.IsNull() && plan.SSHPubKey.ValueString() != "" {
		createReq.SSHPubKey = plan.SSHPubKey.ValueString()
	}

	if !plan.Groups.IsNull() {
		var groups []int64
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
		intGroups := make([]int, len(groups))
		for i, g := range groups {
			intGroups[i] = int(g)
		}
		sort.Ints(intGroups)
		createReq.Groups = intGroups
	}

	if !plan.SudoCommands.IsNull() {
		var cmds []string
		resp.Diagnostics.Append(plan.SudoCommands.ElementsAs(ctx, &cmds, false)...)
		createReq.SudoCommands = cmds
	}

	tflog.Debug(ctx, "Creating user", map[string]interface{}{
		"username": plan.Username.ValueString(),
	})

	user, err := r.client.CreateUser(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating User",
			fmt.Sprintf("Could not create user %q: %s", plan.Username.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, user, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create User success")
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read User start")

	var state UserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse user ID: %s", err))
		return
	}

	user, err := r.client.GetUser(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading User",
			fmt.Sprintf("Could not read user %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, user, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read User success")
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update User start")

	var plan UserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse user ID: %s", err))
		return
	}

	locked := plan.Locked.ValueBool()
	smb := plan.SMB.ValueBool()

	updateReq := &truenas.UserUpdateRequest{
		FullName: plan.FullName.ValueString(),
		Locked:   &locked,
		SMB:      &smb,
	}

	// Three states, and the difference matters: an omitted key leaves the
	// stored hash alone, an explicit null wipes it. Omitting the key while
	// claiming password_disabled would leave the old password working.
	updatePasswordDisabled := plan.PasswordDisabled.ValueBool()
	updateReq.PasswordDisabled = &updatePasswordDisabled
	// Always sent on 26+, never omitted. Omitting it is exactly what let
	// handle_webshare re-add the group from the stale pre-update value.
	if err := setUserWebshare(ctx, r.client, plan.Webshare, func(b *bool) { updateReq.Webshare = b }); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("webshare"), "Invalid Webshare", err.Error())
		return
	}
	switch {
	case updatePasswordDisabled:
		updateReq.ClearPassword()
	case !plan.Password.IsNull() && !plan.Password.IsUnknown():
		updateReq.SetPassword(plan.Password.ValueString())
	}
	if !plan.Email.IsNull() && plan.Email.ValueString() != "" {
		updateReq.Email = plan.Email.ValueString()
	}
	if !plan.Group.IsNull() && !plan.Group.IsUnknown() {
		updateReq.Group = int(plan.Group.ValueInt64())
	}
	if !plan.Home.IsNull() {
		updateReq.Home = plan.Home.ValueString()
	}
	if !plan.Shell.IsNull() {
		updateReq.Shell = plan.Shell.ValueString()
	}
	if !plan.SSHPubKey.IsNull() && plan.SSHPubKey.ValueString() != "" {
		updateReq.SSHPubKey = plan.SSHPubKey.ValueString()
	}

	if !plan.Groups.IsNull() {
		var groups []int64
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
		intGroups := make([]int, len(groups))
		for i, g := range groups {
			intGroups[i] = int(g)
		}
		sort.Ints(intGroups)
		updateReq.Groups = intGroups
	}

	if !plan.SudoCommands.IsNull() {
		var cmds []string
		resp.Diagnostics.Append(plan.SudoCommands.ElementsAs(ctx, &cmds, false)...)
		updateReq.SudoCommands = cmds
	}

	user, err := r.client.UpdateUser(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating User",
			fmt.Sprintf("Could not update user %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, user, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update User success")
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete User start")

	var state UserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse user ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting user", map[string]interface{}{"id": id})

	err = r.client.DeleteUser(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "User already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting User",
			fmt.Sprintf("Could not delete user %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete User success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *UserResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_user")
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	planDisabled := plan.PasswordDisabled.ValueBool()
	planPasswordSet := !plan.Password.IsNull() && !plan.Password.IsUnknown()

	// Create: middleware answers this with "Password is required"
	// (account.py). Catching it at plan time means the operator sees it
	// before anything is created.
	if req.State.Raw.IsNull() {
		if !planPasswordSet && !planDisabled {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Invalid Password",
				"Either set password, or set password_disabled = true for an account "+
					"that should have no password login.",
			)
		}
		return
	}

	// Update: re-enabling password auth without supplying one would send
	// password_disabled=false with no password, leaving an account that
	// claims password login but has unixhash '*'.
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if state.PasswordDisabled.ValueBool() && !planDisabled && !planPasswordSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Invalid Password",
			"This account currently has password_disabled = true. Turning it back off "+
				"requires setting password at the same time.",
		)
	}
}

// ValidateConfig rejects, at plan time, the two combinations middleware
// refuses server-side. Doing it here means the operator gets an attribute
// path instead of an opaque validation error from the API.
func (r *UserResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if !config.PasswordDisabled.ValueBool() {
		return
	}

	// account.py: 'Leave "Password" blank when "Disable password login" is checked.'
	if !config.Password.IsNull() && !config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Invalid Password",
			"password cannot be set when password_disabled = true.",
		)
	}

	// account.py: 'Password authentication may not be disabled for SMB users.'
	// Only an explicit smb = true conflicts; leaving it unset takes the
	// schema default of false and is fine.
	if !config.SMB.IsNull() && !config.SMB.IsUnknown() && config.SMB.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password_disabled"),
			"Invalid Password Disabled",
			"password_disabled cannot be true for an SMB user; TrueNAS requires a password "+
				"to compute the NT hash SMB authentication needs.",
		)
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("User ID must be numeric: %s", err))
		return
	}
	// No password seed. Writing "" here used to fail LengthAtLeast(1) on
	// the next plan, so importing a passwordless account forced the
	// operator to invent a password for it. Leaving it null lets the
	// follow-up Read fill password_disabled from the wire instead.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *UserResource) mapResponseToModel(_ context.Context, user *truenas.User, model *UserResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(user.ID))
	model.UID = types.Int64Value(int64(user.UID))
	model.Username = types.StringValue(user.Username)
	model.FullName = types.StringValue(user.FullName)

	if user.Email != nil {
		model.Email = types.StringValue(*user.Email)
	} else {
		model.Email = types.StringValue("")
	}

	model.Home = types.StringValue(user.Home)
	model.Shell = types.StringValue(user.Shell)
	model.Locked = types.BoolValue(user.Locked)
	model.PasswordDisabled = types.BoolValue(user.PasswordDisabled)
	// A server without the field reports false rather than null, so the
	// attribute stays known and does not produce an unknown-value plan on
	// TrueNAS 25.10.
	model.Webshare = types.BoolValue(user.Webshare != nil && *user.Webshare)
	model.SMB = types.BoolValue(user.SMB)
	model.Group = types.Int64Value(int64(user.Group.ID))

	if user.SSHPubKey != nil {
		model.SSHPubKey = types.StringValue(*user.SSHPubKey)
	} else {
		model.SSHPubKey = types.StringValue("")
	}

	// groups: unordered set; sorted only so logs and API payloads are deterministic
	ids := make([]int, len(user.Groups))
	copy(ids, user.Groups)
	sort.Ints(ids)
	groupValues := make([]attr.Value, len(ids))
	for i, g := range ids {
		groupValues[i] = types.Int64Value(int64(g))
	}
	model.Groups = types.SetValueMust(types.Int64Type, groupValues)

	// sudo_commands
	cmdValues := make([]attr.Value, len(user.SudoCommands))
	for i, c := range user.SudoCommands {
		cmdValues[i] = types.StringValue(c)
	}
	model.SudoCommands = types.ListValueMust(types.StringType, cmdValues)
}

// setUserWebshare applies the webshare attribute to a create or update
// request, but only against a server that has the field.
//
// webshare arrived in TrueNAS 26.0. Both the 25.10 and 26.0 models are
// ConfigDict(extra="forbid"), so sending the key to 25.10 is a hard
// ValidationError, not an ignored field. Leaving it unset on 26+ is equally
// wrong: handle_webshare then inherits the pre-update value and re-adds the
// group, which is the bug this attribute exists to fix. So it is always sent
// on 26+, and never sent below that.
//
// A user who explicitly asks for webshare on an older server gets a clear
// refusal rather than a pydantic error from middleware.
func setUserWebshare(ctx context.Context, c *wsclient.Client, planned types.Bool, apply func(*bool)) error {
	v, err := c.ServerVersion(ctx)
	if err != nil {
		return err
	}
	if v.AtLeast(26, 0) {
		b := planned.ValueBool()
		apply(&b)
		return nil
	}
	if !planned.IsNull() && !planned.IsUnknown() && planned.ValueBool() {
		return fmt.Errorf("webshare requires TrueNAS 26.0 or newer; this server reports %s", v)
	}
	return nil
}
