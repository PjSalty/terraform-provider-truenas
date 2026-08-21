package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                = &SMBConfigResource{}
	_ resource.ResourceWithImportState = &SMBConfigResource{}
	_ resource.ResourceWithModifyPlan  = &SMBConfigResource{}
)

// SMBConfigResource manages the TrueNAS SMB service configuration.
type SMBConfigResource struct {
	client *wsclient.Client
}

// SMBConfigResourceModel describes the resource data model.
type SMBConfigResourceModel struct {
	ID              types.String `tfsdk:"id"`
	NetbiosName     types.String `tfsdk:"netbiosname"`
	Workgroup       types.String `tfsdk:"workgroup"`
	Description     types.String `tfsdk:"description"`
	EnableSMB1      types.Bool   `tfsdk:"enable_smb1"`
	MinimumProtocol types.String `tfsdk:"minimum_protocol"`
	SearchProtocols types.List   `tfsdk:"search_protocols"`

	UnixCharset    types.String   `tfsdk:"unixcharset"`
	AAPLExtensions types.Bool     `tfsdk:"aapl_extensions"`
	Guest          types.String   `tfsdk:"guest"`
	Filemask       types.String   `tfsdk:"filemask"`
	Dirmask        types.String   `tfsdk:"dirmask"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func NewSMBConfigResource() resource.Resource {
	return &SMBConfigResource{}
}

func (r *SMBConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smb_config"
}

func (r *SMBConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the SMB service configuration on TrueNAS SCALE. " +
		"This is a singleton resource, only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"netbiosname": schema.StringAttribute{
				Description: "NetBIOS name of the server.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("truenas"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 15),
				},
			},
			"workgroup": schema.StringAttribute{
				Description: "Windows workgroup name.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("WORKGROUP"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 15),
				},
			},
			"description": schema.StringAttribute{
				Description: "Server description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("TrueNAS Server"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			// No Default here, deliberately. booldefault.StaticBool(false)
			// made the planned value always known-and-false, so
			// buildUpdateRequest emitted enable_smb1 on EVERY apply,
			// including the reset path, for users who never set it. On
			// TrueNAS 26.0 that key is a hard ValidationError, so the
			// static default is what turned "user does not care about
			// SMB1" into "every apply fails". UseStateForUnknown supplies
			// the refreshed server value instead.
			"enable_smb1": schema.BoolAttribute{
				Description: "Enable SMB1 protocol support. Deprecated: use `minimum_protocol` instead. " +
					"`true` is equivalent to `minimum_protocol = \"SMB1\"`, `false` to `\"SMB2\"`.",
				DeprecationMessage: "Use minimum_protocol instead. TrueNAS 26.0 replaced enable_smb1 with " +
					"minimum_protocol; set minimum_protocol = \"SMB1\" or \"SMB2\".",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.MatchRoot("minimum_protocol")),
				},
			},
			// The going-forward attribute. Works on every supported
			// version: the client translates SMB1/SMB2 back to the legacy
			// boolean on TrueNAS 25.10 and older.
			"minimum_protocol": schema.StringAttribute{
				Description: "Minimum SMB protocol version the server will negotiate " +
					"(`SMB1`, `SMB2`, or `SMB3`). `SMB3` requires TrueNAS 26.0 or newer.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(truenas.SMBProtocolSMB1, truenas.SMBProtocolSMB2, truenas.SMBProtocolSMB3),
					stringvalidator.ConflictsWith(path.MatchRoot("enable_smb1")),
				},
			},
			// New in TrueNAS 26.0. Version-gated the same way as
			// minimum_protocol: the pre-26 models are extra="forbid", so the
			// key must not reach an older server.
			"search_protocols": schema.ListAttribute{
				Description: "Extra search protocols the SMB server answers. Currently only " +
					"`SPOTLIGHT` (macOS Spotlight). Requires TrueNAS 26.0 or newer.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("SPOTLIGHT")),
				},
			},
			"unixcharset": schema.StringAttribute{
				Description: "UNIX character set.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("UTF-8"),
				Validators: []validator.String{
					stringvalidator.OneOf("UTF-8", "ISO-8859-1", "ISO-8859-15", "GB2312", "EUC-JP", "ASCII"),
				},
			},
			"aapl_extensions": schema.BoolAttribute{
				Description: "Enable Apple SMB2/3 protocol extensions.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"guest": schema.StringAttribute{
				Description: "Guest account for unauthenticated access.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("nobody"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
				},
			},
			"filemask": schema.StringAttribute{
				Description: "File creation mask.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DEFAULT"),
			},
			"dirmask": schema.StringAttribute{
				Description: "Directory creation mask.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DEFAULT"),
			},
		},
	}
}

func (r *SMBConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SMBConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SMBConfig start")

	var plan SMBConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SMB config resource (updating singleton)")

	updateReq := r.buildUpdateRequest(ctx, &plan)

	config, err := r.client.UpdateSMBConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SMB Config",
			fmt.Sprintf("Could not update SMB configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SMBConfig success")
}

func (r *SMBConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SMBConfig start")

	var state SMBConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetSMBConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SMB Config",
			fmt.Sprintf("Could not read SMB configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SMBConfig success")
}

func (r *SMBConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SMBConfig start")

	var plan SMBConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(ctx, &plan)

	config, err := r.client.UpdateSMBConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SMB Config",
			fmt.Sprintf("Could not update SMB configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SMBConfig success")
}

func (r *SMBConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SMBConfig start")

	tflog.Debug(ctx, "Deleting SMB config resource (resetting to defaults)")

	netbiosname := "truenas"
	workgroup := "WORKGROUP"
	description := "TrueNAS Server"
	// SMB2 is middleware's own default for the new column (plugins/smb.py)
	// and is what the legacy enable_smb1=false mapped to, so this resets to
	// exactly the same state as before on 25.10 while also being valid on 26.
	minProto := truenas.SMBProtocolSMB2
	unixcharset := "UTF-8"
	aaplExtensions := false
	guest := "nobody"
	filemask := "DEFAULT"
	dirmask := "DEFAULT"

	_, err := r.client.UpdateSMBConfig(ctx, &truenas.SMBConfigUpdateRequest{
		NetbiosName:     &netbiosname,
		Workgroup:       &workgroup,
		Description:     &description,
		MinimumProtocol: &minProto,
		UnixCharset:     &unixcharset,
		AAPLExtensions:  &aaplExtensions,
		Guest:           &guest,
		Filemask:        &filemask,
		Dirmask:         &dirmask,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting SMB Config",
			fmt.Sprintf("Could not reset SMB configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete SMBConfig success")
}

func (r *SMBConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan keeps enable_smb1 and minimum_protocol in lockstep so that
// whichever one the user declared, the plan carries a concrete value for
// both. buildUpdateRequest then only ever reads minimum_protocol.
//
// Both attributes are Optional+Computed, so a user who declares neither gets
// the refreshed server values via UseStateForUnknown and nothing here fires.
// The validators make declaring both an error, so the two branches below are
// genuinely exclusive.
func (r *SMBConfigResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy: there is no plan to reconcile.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Diagnostics are appended, not short-circuited on: matching the house
	// pattern in certificate.go. A failed Get leaves both fields null, so
	// the switch below falls through and the framework still surfaces the
	// error it just recorded.
	var config SMBConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	switch {
	case !config.MinimumProtocol.IsNull() && !config.MinimumProtocol.IsUnknown():
		// Going-forward attribute wins; derive the deprecated one so state
		// stays self-consistent.
		smb1 := config.MinimumProtocol.ValueString() == truenas.SMBProtocolSMB1
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("enable_smb1"), types.BoolValue(smb1))...)

	case !config.EnableSMB1.IsNull() && !config.EnableSMB1.IsUnknown():
		// Deprecated attribute declared: translate it forward using the
		// same mapping middleware's own migration uses
		// (true -> SMB1, false -> SMB2).
		proto := truenas.SMBProtocolSMB2
		if config.EnableSMB1.ValueBool() {
			proto = truenas.SMBProtocolSMB1
		}
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("minimum_protocol"), types.StringValue(proto))...)
	}
}

func (r *SMBConfigResource) buildUpdateRequest(ctx context.Context, plan *SMBConfigResourceModel) *truenas.SMBConfigUpdateRequest {
	updateReq := &truenas.SMBConfigUpdateRequest{}

	if !plan.NetbiosName.IsNull() && !plan.NetbiosName.IsUnknown() {
		v := plan.NetbiosName.ValueString()
		updateReq.NetbiosName = &v
	}
	if !plan.Workgroup.IsNull() && !plan.Workgroup.IsUnknown() {
		v := plan.Workgroup.ValueString()
		updateReq.Workgroup = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	// Only the normalized attribute is ever sent. ModifyPlan has already
	// derived minimum_protocol from enable_smb1 when the user declared the
	// deprecated one, so this single branch covers all three cases and the
	// legacy boolean never reaches the wire encoder.
	if !plan.MinimumProtocol.IsNull() && !plan.MinimumProtocol.IsUnknown() {
		v := plan.MinimumProtocol.ValueString()
		updateReq.MinimumProtocol = &v
	}
	if !plan.SearchProtocols.IsNull() && !plan.SearchProtocols.IsUnknown() {
		// A nil slice here would marshal as JSON null, which middleware
		// rejects with "Input should be a valid list". The wire encoder in
		// wsclient normalises that in one place for every caller, so this
		// just forwards what the plan holds.
		var got []string
		plan.SearchProtocols.ElementsAs(ctx, &got, false)
		updateReq.SearchProtocols = &got
	}
	if !plan.UnixCharset.IsNull() && !plan.UnixCharset.IsUnknown() {
		v := plan.UnixCharset.ValueString()
		updateReq.UnixCharset = &v
	}
	if !plan.AAPLExtensions.IsNull() && !plan.AAPLExtensions.IsUnknown() {
		v := plan.AAPLExtensions.ValueBool()
		updateReq.AAPLExtensions = &v
	}
	if !plan.Guest.IsNull() && !plan.Guest.IsUnknown() {
		v := plan.Guest.ValueString()
		updateReq.Guest = &v
	}
	if !plan.Filemask.IsNull() && !plan.Filemask.IsUnknown() {
		v := plan.Filemask.ValueString()
		updateReq.Filemask = &v
	}
	if !plan.Dirmask.IsNull() && !plan.Dirmask.IsUnknown() {
		v := plan.Dirmask.ValueString()
		updateReq.Dirmask = &v
	}

	return updateReq
}

func (r *SMBConfigResource) mapResponseToModel(ctx context.Context, config *truenas.SMBConfig, model *SMBConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.NetbiosName = types.StringValue(config.NetbiosName)
	model.Workgroup = types.StringValue(config.Workgroup)
	model.Description = types.StringValue(config.Description)
	// Both are written from the SAME normalized source, so the deprecated
	// attribute and its replacement can never disagree, on either server
	// version, on any of Create/Read/Update/Import.
	model.MinimumProtocol = types.StringValue(config.Protocol)
	// A server without the field reports an empty list rather than null, so
	// the attribute stays known and plans on 25.10 show no phantom diff.
	sp := []string{}
	if config.SearchProtocols != nil {
		sp = *config.SearchProtocols
	}
	spVal, _ := types.ListValueFrom(ctx, types.StringType, sp)
	model.SearchProtocols = spVal
	model.EnableSMB1 = types.BoolValue(config.SMB1Enabled)
	model.UnixCharset = types.StringValue(config.UnixCharset)
	model.AAPLExtensions = types.BoolValue(config.AAPLExtensions)
	model.Guest = types.StringValue(config.Guest)
	model.Filemask = types.StringValue(config.Filemask)
	model.Dirmask = types.StringValue(config.Dirmask)
}
