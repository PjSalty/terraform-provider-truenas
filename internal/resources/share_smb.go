package resources

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                   = &SMBShareResource{}
	_ resource.ResourceWithImportState    = &SMBShareResource{}
	_ resource.ResourceWithModifyPlan     = &SMBShareResource{}
	_ resource.ResourceWithValidateConfig = &SMBShareResource{}
)

// smbOptionPurposes maps each options attribute to the share purposes that
// accept it. TrueNAS models options as a discriminated union on purpose and
// rejects a field belonging to another member outright, so a wrong pairing is
// an error rather than something quietly ignored:
//
//	[EINVAL] data.options.DEFAULT_SHARE.remote_path: Extra inputs are not permitted
//
// Checking it here turns that into a plan-time failure naming the attribute
// and the purposes it belongs to. Verified against 26.0-BETA.1.
const (
	smbPurposeExternal = "EXTERNAL_SHARE"
	smbExternalPath    = "EXTERNAL"
	smbGracePeriodMin  = 60
	smbGracePeriodMax  = 86400 * 180
)

// smbPurposes is the accepted purpose vocabulary, declared once so the schema
// validator and smbOptionPurposes cannot drift apart. Verified against the
// upstream SMBSharePurpose enum for 25.10.x and 26.0.
var smbPurposes = []string{
	"DEFAULT_SHARE",
	"LEGACY_SHARE",
	"TIMEMACHINE_SHARE",
	"MULTIPROTOCOL_SHARE",
	"PRIVATE_DATASETS_SHARE",
	"EXTERNAL_SHARE",
	"TIME_LOCKED_SHARE",
	"VEEAM_REPOSITORY_SHARE",
	"FCP_SHARE",
}

var smbOptionPurposes = map[string][]string{
	"remote_path":           {"EXTERNAL_SHARE"},
	"aapl_name_mangling":    {"DEFAULT_SHARE", "LEGACY_SHARE", "MULTIPROTOCOL_SHARE", "TIME_LOCKED_SHARE", "PRIVATE_DATASETS_SHARE", "FCP_SHARE"},
	"timemachine_quota":     {"TIMEMACHINE_SHARE", "LEGACY_SHARE"},
	"auto_snapshot":         {"TIMEMACHINE_SHARE"},
	"auto_dataset_creation": {"TIMEMACHINE_SHARE"},
	"dataset_naming_schema": {"TIMEMACHINE_SHARE", "PRIVATE_DATASETS_SHARE"},
	"auto_quota":            {"PRIVATE_DATASETS_SHARE"},
	"grace_period":          {"TIME_LOCKED_SHARE"},
	"vuid":                  {"TIMEMACHINE_SHARE", "LEGACY_SHARE"},
	// hostsallow and hostsdeny are accepted by every purpose except
	// EXTERNAL_SHARE, which takes remote_path and nothing else.
	"hostsallow": {"DEFAULT_SHARE", "LEGACY_SHARE", "TIMEMACHINE_SHARE", "MULTIPROTOCOL_SHARE", "TIME_LOCKED_SHARE", "PRIVATE_DATASETS_SHARE", "VEEAM_REPOSITORY_SHARE", "FCP_SHARE"},
	"hostsdeny":  {"DEFAULT_SHARE", "LEGACY_SHARE", "TIMEMACHINE_SHARE", "MULTIPROTOCOL_SHARE", "TIME_LOCKED_SHARE", "PRIVATE_DATASETS_SHARE", "VEEAM_REPOSITORY_SHARE", "FCP_SHARE"},
}

// smbOptionAttrTypes is declared once so the schema, the state writes and the
// tests cannot drift apart.
var smbOptionAttrTypes = map[string]attr.Type{
	"remote_path":           types.ListType{ElemType: types.StringType},
	"hostsallow":            types.ListType{ElemType: types.StringType},
	"hostsdeny":             types.ListType{ElemType: types.StringType},
	"aapl_name_mangling":    types.BoolType,
	"timemachine_quota":     types.Int64Type,
	"auto_snapshot":         types.BoolType,
	"auto_dataset_creation": types.BoolType,
	"dataset_naming_schema": types.StringType,
	"auto_quota":            types.Int64Type,
	"grace_period":          types.Int64Type,
	"vuid":                  types.StringType,
}

// SMBShareResource manages a TrueNAS SMB share.
type SMBShareResource struct {
	client *wsclient.Client
}

// SMBShareResourceModel describes the resource data model.
type SMBShareResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Path      types.String   `tfsdk:"path"`
	Name      types.String   `tfsdk:"name"`
	Comment   types.String   `tfsdk:"comment"`
	Browsable types.Bool     `tfsdk:"browsable"`
	ReadOnly  types.Bool     `tfsdk:"readonly"`
	ABE       types.Bool     `tfsdk:"abe"`
	Enabled   types.Bool     `tfsdk:"enabled"`
	Purpose   types.String   `tfsdk:"purpose"`
	Options   types.Object   `tfsdk:"options"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

func NewSMBShareResource() resource.Resource {
	return &SMBShareResource{}
}

func (r *SMBShareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_smb"
}

func (r *SMBShareResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an SMB share on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the SMB share.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path to share (e.g., /mnt/tank/data). Must start with /mnt/, " +
					"except for an EXTERNAL_SHARE, where it is the literal string EXTERNAL " +
					"because the share proxies to `options.remote_path` rather than serving " +
					"anything local.",
				Required: true,
				Validators: []validator.String{
					// LengthAtMost, not LengthBetween(5, ...): the regex below
					// already requires either the 5-character /mnt/ prefix or
					// the 8-character sentinel, so a lower bound can never fire
					// and only reads as a check that is not there.
					stringvalidator.LengthAtMost(1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(?:/mnt/.*|`+smbExternalPath+`)$`),
						"SMB share path must start with /mnt/, or be the literal string "+
							smbExternalPath+" for an EXTERNAL_SHARE",
					),
				},
			},
			"name": schema.StringAttribute{
				Description: "The share name visible to SMB clients (1-80 chars, no slashes).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 80),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[^/\\:*?"<>|]+$`),
						"SMB share name cannot contain / \\ : * ? \" < > |",
					),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment describing the share.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"browsable": schema.BoolAttribute{
				Description: "Whether the share is browsable in network discovery.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"readonly": schema.BoolAttribute{
				Description: "Whether the share is read-only.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"abe": schema.BoolAttribute{
				Description: "Whether Access Based Share Enumeration is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the share is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"purpose": schema.StringAttribute{
				Description: "The share purpose preset. The valid value set is " +
					"the TrueNAS SCALE 25.10+ preset vocabulary. The earlier " +
					"vocabulary (ENHANCED_TIMEMACHINE, LEGACY_SMB_WHITELIST, " +
					"MULTI_PROTOCOL_NFS, MULTI_PROTOCOL_AFP, PRIVATE_DATASETS, " +
					"NO_PRESET, TIMEMACHINE) was retired in the SMB preset overhaul " +
					"and no longer accepted by the upstream API. FCP_SHARE, for " +
					"Final Cut Pro storage, was added in 25.10.1, so it is rejected " +
					"by a 25.10.0 server. Each purpose accepts a different set of " +
					"`options`; see that attribute. TIMEMACHINE_SHARE and FCP_SHARE " +
					"additionally require `aapl_extensions` on the global SMB config " +
					"(`truenas_smb_config`); without it the create fails with an " +
					"EINVAL naming `purpose` rather than the setting it wants.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(smbPurposes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"options": schema.SingleNestedAttribute{
				Description: "Purpose-specific settings. TrueNAS models these as a union keyed on `purpose`, " +
					"so an attribute belonging to a different purpose is rejected rather than ignored; each " +
					"one below names the purposes that accept it. Omit the block to take the purpose's " +
					"defaults, except for EXTERNAL_SHARE, which has no usable default and requires remote_path.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"remote_path": schema.ListAttribute{
						Description: "EXTERNAL_SHARE only, and required for it. DFS proxy targets, each `SERVER\\SHARE`, " +
							"where SERVER is a full domain name or IP. TrueNAS does not check that they are reachable.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"hostsallow": schema.ListAttribute{
						Description: "Hosts permitted to access the share. Every purpose except EXTERNAL_SHARE.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"hostsdeny": schema.ListAttribute{
						Description: "Hosts denied access to the share. Every purpose except EXTERNAL_SHARE.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"aapl_name_mangling": schema.BoolAttribute{
						Description: "Store the illegal-NTFS characters macOS clients use with their native values. " +
							"Do not change once data is written. Non-macOS clients may not see such files.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"timemachine_quota": schema.Int64Attribute{
						Description: "TIMEMACHINE_SHARE. Maximum size in bytes reported to the client for one " +
							"sparsebundle. 0 means no quota. Modern macOS sets this client-side, which behaves " +
							"more predictably.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"auto_snapshot": schema.BoolAttribute{
						Description: "TIMEMACHINE_SHARE. Snapshot the share dataset when a client starts a new backup.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"auto_dataset_creation": schema.BoolAttribute{
						Description: "TIMEMACHINE_SHARE. Create a dataset per connecting user.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"dataset_naming_schema": schema.StringAttribute{
						Description: "TIMEMACHINE_SHARE and PRIVATE_DATASETS_SHARE. Naming schema for per-user " +
							"datasets, for example `%D/%U`. Unset means `%U`, or `%D/%U` when joined to Active " +
							"Directory. ZFS naming rules are stricter than path rules.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"auto_quota": schema.Int64Attribute{
						Description: "PRIVATE_DATASETS_SHARE. Quota in gibibytes applied to new datasets. 0 means none.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"grace_period": schema.Int64Attribute{
						Description: "TIME_LOCKED_SHARE. Seconds during which a new file or directory stays writable. " +
							"Between 60 and 15552000.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vuid": schema.StringAttribute{
						Description: "TIMEMACHINE_SHARE and LEGACY_SHARE. Volume UUID advertised to clients.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *SMBShareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SMBShareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SMBShare start")

	var plan SMBShareResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	createReq := &truenas.SMBShareCreateRequest{
		Path:      plan.Path.ValueString(),
		Name:      plan.Name.ValueString(),
		Browsable: plan.Browsable.ValueBool(),
		ReadOnly:  plan.ReadOnly.ValueBool(),
		ABE:       plan.ABE.ValueBool(),
		Enabled:   plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() {
		createReq.Comment = plan.Comment.ValueString()
	}
	if !plan.Purpose.IsNull() {
		createReq.Purpose = plan.Purpose.ValueString()
	}
	opts, optErr := smbOptionsFromModel(ctx, plan.Options)
	if optErr != nil {
		resp.Diagnostics.AddError("Error Creating SMB Share", optErr.Error())
		return
	}
	createReq.Options = opts

	tflog.Debug(ctx, "Creating SMB share", map[string]interface{}{
		"name": plan.Name.ValueString(),
		"path": plan.Path.ValueString(),
	})

	share, err := r.client.CreateSMBShare(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating SMB Share",
			fmt.Sprintf("Could not create SMB share %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(share, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SMBShare success")
}

func (r *SMBShareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SMBShare start")

	var state SMBShareResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse SMB share ID: %s", err))
		return
	}

	share, err := r.client.GetSMBShare(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading SMB Share",
			fmt.Sprintf("Could not read SMB share %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(share, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SMBShare success")
}

func (r *SMBShareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SMBShare start")

	var plan SMBShareResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	var state SMBShareResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse SMB share ID: %s", err))
		return
	}

	browsable := plan.Browsable.ValueBool()
	readOnly := plan.ReadOnly.ValueBool()
	abe := plan.ABE.ValueBool()
	enabled := plan.Enabled.ValueBool()

	updateReq := &truenas.SMBShareUpdateRequest{
		Path:      plan.Path.ValueString(),
		Name:      plan.Name.ValueString(),
		Browsable: &browsable,
		ReadOnly:  &readOnly,
		ABE:       &abe,
		Enabled:   &enabled,
	}

	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}
	if !plan.Purpose.IsNull() {
		updateReq.Purpose = plan.Purpose.ValueString()
	}
	updOpts, optErr := smbOptionsFromModel(ctx, plan.Options)
	if optErr != nil {
		resp.Diagnostics.AddError("Error Updating SMB Share", optErr.Error())
		return
	}
	updateReq.Options = updOpts

	share, err := r.client.UpdateSMBShare(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SMB Share",
			fmt.Sprintf("Could not update SMB share %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(share, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SMBShare success")
}

func (r *SMBShareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SMBShare start")

	var state SMBShareResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse SMB share ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting SMB share", map[string]interface{}{
		"id": id,
	})

	err = r.client.DeleteSMBShare(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "SMB share already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting SMB Share",
			fmt.Sprintf("Could not delete SMB share %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete SMBShare success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *SMBShareResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_share_smb")
}

func (r *SMBShareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("SMB share ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SMBShareResource) mapResponseToModel(share *truenas.SMBShare, model *SMBShareResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(share.ID))
	model.Path = types.StringValue(share.Path)
	model.Name = types.StringValue(share.Name)
	model.Comment = types.StringValue(share.Comment)
	model.Browsable = types.BoolValue(share.Browsable)
	model.ReadOnly = types.BoolValue(share.ReadOnly)
	model.ABE = types.BoolValue(share.ABE)
	model.Enabled = types.BoolValue(share.Enabled)
	model.Purpose = types.StringValue(share.Purpose)
	model.Options = smbOptionsToObject(share.Options)
}

// ValidateConfig enforces the options union at plan time.
//
// TrueNAS keys options on the share's purpose and rejects a field belonging to
// another member, so a wrong pairing is a hard error rather than something
// ignored. Catching it here names the attribute and the purposes that accept
// it, instead of surfacing as:
//
//	[EINVAL] data.options.DEFAULT_SHARE.remote_path: Extra inputs are not permitted
func (r *SMBShareResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg SMBShareResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// purpose defaults to DEFAULT_SHARE server-side when unset.
	purpose := "DEFAULT_SHARE"
	if !cfg.Purpose.IsNull() && !cfg.Purpose.IsUnknown() && cfg.Purpose.ValueString() != "" {
		purpose = cfg.Purpose.ValueString()
	}

	// EXTERNAL_SHARE has no usable default: middleware answers
	// "remote_path: Field required" when options are omitted, and insists the
	// path is the literal string EXTERNAL.
	if purpose == smbPurposeExternal {
		if cfg.Options.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("options"), "Invalid SMB Share",
				"EXTERNAL_SHARE requires an options block with remote_path. It is a DFS proxy, so there "+
					"is no local path to fall back on and TrueNAS rejects the share outright without it.")
		}
		if !cfg.Path.IsNull() && !cfg.Path.IsUnknown() && cfg.Path.ValueString() != smbExternalPath {
			resp.Diagnostics.AddAttributeError(path.Root("path"), "Invalid SMB Share",
				fmt.Sprintf("EXTERNAL_SHARE requires path to be the literal string %q, got %q. The share "+
					"proxies to remote_path rather than serving anything local.",
					smbExternalPath, cfg.Path.ValueString()))
		}
	}

	// The path validator admits EXTERNAL so an EXTERNAL_SHARE can be written
	// at all. Only that purpose may use it: on any other, TrueNAS treats it as
	// a local path and fails with a mount error that names neither field.
	if purpose != smbPurposeExternal &&
		!cfg.Path.IsNull() && !cfg.Path.IsUnknown() && cfg.Path.ValueString() == smbExternalPath {
		resp.Diagnostics.AddAttributeError(path.Root("path"), "Invalid SMB Share",
			fmt.Sprintf("path %q is only valid for purpose %s, not %s. Every other purpose "+
				"serves a local path under /mnt/.", smbExternalPath, smbPurposeExternal, purpose))
	}

	if cfg.Options.IsNull() || cfg.Options.IsUnknown() {
		return
	}

	for name, v := range cfg.Options.Attributes() {
		if v.IsNull() || v.IsUnknown() {
			continue
		}
		// Every options attribute has an entry: TestSMBOptionPurposesCoversSchema
		// fails if one is added to the schema without one, which is what keeps
		// a new attribute from arriving silently unvalidated.
		allowed := smbOptionPurposes[name]
		if slices.Contains(allowed, purpose) {
			continue
		}
		resp.Diagnostics.AddAttributeError(path.Root("options").AtName(name), "Invalid SMB Share",
			fmt.Sprintf("%s does not apply to purpose %s; TrueNAS accepts it only on %s. "+
				"Options are keyed on the purpose, so an attribute from another one is rejected, not ignored.",
				name, purpose, strings.Join(allowed, ", ")))
	}

	// remote_path entries are SERVER\SHARE. Middleware rejects anything else
	// with "DFS proxy must be of format SERVER\SHARE", which is easy to hit
	// by writing a URL or a bare hostname.
	if rp, ok := cfg.Options.Attributes()["remote_path"].(types.List); ok && !rp.IsNull() && !rp.IsUnknown() {
		for i, elem := range rp.Elements() {
			sv, ok := elem.(types.String)
			if !ok || sv.IsNull() || sv.IsUnknown() {
				continue
			}
			if !strings.Contains(sv.ValueString(), `\`) {
				resp.Diagnostics.AddAttributeError(
					path.Root("options").AtName("remote_path").AtListIndex(i), "Invalid SMB Share",
					fmt.Sprintf("remote_path entries are SERVER\\SHARE, for example %s, got %q.",
						`192.168.0.200\SHARE`, sv.ValueString()))
			}
		}
	}

	if gp, ok := cfg.Options.Attributes()["grace_period"].(types.Int64); ok && !gp.IsNull() && !gp.IsUnknown() {
		if v := gp.ValueInt64(); v < smbGracePeriodMin || v > smbGracePeriodMax {
			resp.Diagnostics.AddAttributeError(path.Root("options").AtName("grace_period"), "Invalid SMB Share",
				fmt.Sprintf("grace_period must be between %d and %d seconds, got %d.",
					smbGracePeriodMin, smbGracePeriodMax, v))
		}
	}
}

// smbOptionsFromModel turns the options block into the wire union. Only
// attributes the configuration actually set are sent: an omitted one must stay
// omitted so the purpose's own default applies.
func smbOptionsFromModel(ctx context.Context, o types.Object) (*truenas.SMBShareOptions, error) {
	if o.IsNull() || o.IsUnknown() {
		return nil, nil
	}
	attrs := o.Attributes()
	out := &truenas.SMBShareOptions{}

	strList := func(name string) (*[]string, error) {
		v, ok := attrs[name].(types.List)
		if !ok || v.IsNull() || v.IsUnknown() {
			return nil, nil
		}
		// ElementsAs yields a nil slice for an empty list, and a non-nil
		// pointer to a nil slice marshals as JSON null, which middleware
		// rejects as not-a-list. So it is always allocated first.
		out := make([]string, 0, len(v.Elements()))
		if diags := v.ElementsAs(ctx, &out, false); diags.HasError() {
			return nil, fmt.Errorf("reading options.%s: %v", name, diags.Errors())
		}
		return &out, nil
	}
	boolPtr := func(name string) *bool {
		v, ok := attrs[name].(types.Bool)
		if !ok || v.IsNull() || v.IsUnknown() {
			return nil
		}
		b := v.ValueBool()
		return &b
	}
	intPtr := func(name string) *int64 {
		v, ok := attrs[name].(types.Int64)
		if !ok || v.IsNull() || v.IsUnknown() {
			return nil
		}
		i := v.ValueInt64()
		return &i
	}
	strPtr := func(name string) *string {
		v, ok := attrs[name].(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			return nil
		}
		s := v.ValueString()
		return &s
	}

	var err error
	if out.RemotePath, err = strList("remote_path"); err != nil {
		return nil, err
	}
	if out.HostsAllow, err = strList("hostsallow"); err != nil {
		return nil, err
	}
	if out.HostsDeny, err = strList("hostsdeny"); err != nil {
		return nil, err
	}
	out.AAPLNameMangling = boolPtr("aapl_name_mangling")
	out.AutoSnapshot = boolPtr("auto_snapshot")
	out.AutoDatasetCreation = boolPtr("auto_dataset_creation")
	out.TimeMachineQuota = intPtr("timemachine_quota")
	out.AutoQuota = intPtr("auto_quota")
	out.GracePeriod = intPtr("grace_period")
	out.DatasetNamingSchema = strPtr("dataset_naming_schema")
	out.VUID = strPtr("vuid")
	return out, nil
}

// smbOptionsToObject writes the server's view of options back into state.
//
// The server echoes only the fields belonging to the share's purpose, so every
// other attribute is null rather than a zero value. A null says "this purpose
// does not have it", which a zero would misreport as "set to 0" or "off".
func smbOptionsToObject(o *truenas.SMBShareOptions) types.Object {
	if o == nil {
		return types.ObjectNull(smbOptionAttrTypes)
	}
	// Values are built element by element rather than through ListValueFrom,
	// so there is no conversion that can fail and therefore no unreachable
	// error branch to rot. Must panics only on a type mismatch, which cannot
	// happen when every element is a types.StringValue.
	list := func(p *[]string) attr.Value {
		if p == nil {
			return types.ListNull(types.StringType)
		}
		elems := make([]attr.Value, 0, len(*p))
		for _, item := range *p {
			elems = append(elems, types.StringValue(item))
		}
		return types.ListValueMust(types.StringType, elems)
	}
	b := func(p *bool) attr.Value {
		if p == nil {
			return types.BoolNull()
		}
		return types.BoolValue(*p)
	}
	i := func(p *int64) attr.Value {
		if p == nil {
			return types.Int64Null()
		}
		return types.Int64Value(*p)
	}
	sv := func(p *string) attr.Value {
		if p == nil {
			return types.StringNull()
		}
		return types.StringValue(*p)
	}

	return types.ObjectValueMust(smbOptionAttrTypes, map[string]attr.Value{
		"remote_path":           list(o.RemotePath),
		"hostsallow":            list(o.HostsAllow),
		"hostsdeny":             list(o.HostsDeny),
		"aapl_name_mangling":    b(o.AAPLNameMangling),
		"auto_snapshot":         b(o.AutoSnapshot),
		"auto_dataset_creation": b(o.AutoDatasetCreation),
		"timemachine_quota":     i(o.TimeMachineQuota),
		"auto_quota":            i(o.AutoQuota),
		"grace_period":          i(o.GracePeriod),
		"dataset_naming_schema": sv(o.DatasetNamingSchema),
		"vuid":                  sv(o.VUID),
	})
}
