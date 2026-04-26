package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &SMBShareResource{}
	_ resource.ResourceWithImportState = &SMBShareResource{}
	_ resource.ResourceWithModifyPlan  = &SMBShareResource{}
)

// SMBShareResource manages a TrueNAS SMB share.
type SMBShareResource struct {
	client *client.Client
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
				Description: "The path to share (e.g., /mnt/tank/data). Must start with /mnt/.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^/mnt/`),
						"SMB share path must start with /mnt/",
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
				Description: "The share purpose preset.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("DEFAULT_SHARE", "ENHANCED_TIMEMACHINE", "LEGACY_SMB_WHITELIST", "MULTI_PROTOCOL_NFS", "MULTI_PROTOCOL_AFP", "PRIVATE_DATASETS", "NO_PRESET", "TIMEMACHINE"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SMBShareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
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

	createReq := &client.SMBShareCreateRequest{
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

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse SMB share ID: %s", err))
		return
	}

	share, err := r.client.GetSMBShare(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
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

	updateReq := &client.SMBShareUpdateRequest{
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
		if client.IsNotFound(err) {
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

func (r *SMBShareResource) mapResponseToModel(share *client.SMBShare, model *SMBShareResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(share.ID))
	model.Path = types.StringValue(share.Path)
	model.Name = types.StringValue(share.Name)
	model.Comment = types.StringValue(share.Comment)
	model.Browsable = types.BoolValue(share.Browsable)
	model.ReadOnly = types.BoolValue(share.ReadOnly)
	model.ABE = types.BoolValue(share.ABE)
	model.Enabled = types.BoolValue(share.Enabled)
	model.Purpose = types.StringValue(share.Purpose)
}
