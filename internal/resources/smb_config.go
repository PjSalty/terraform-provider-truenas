package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &SMBConfigResource{}
	_ resource.ResourceWithImportState = &SMBConfigResource{}
)

// SMBConfigResource manages the TrueNAS SMB service configuration.
type SMBConfigResource struct {
	client *client.Client
}

// SMBConfigResourceModel describes the resource data model.
type SMBConfigResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	NetbiosName    types.String   `tfsdk:"netbiosname"`
	Workgroup      types.String   `tfsdk:"workgroup"`
	Description    types.String   `tfsdk:"description"`
	EnableSMB1     types.Bool     `tfsdk:"enable_smb1"`
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
		"This is a singleton resource — only one instance can exist.",
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
			"enable_smb1": schema.BoolAttribute{
				Description: "Enable SMB1 protocol support.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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

func (r *SMBConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SMBConfig start")

	var plan SMBConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SMB config resource (updating singleton)")

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateSMBConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SMB Config",
			fmt.Sprintf("Could not update SMB configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

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

	r.mapResponseToModel(config, &state)

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

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateSMBConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SMB Config",
			fmt.Sprintf("Could not update SMB configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

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
	enableSMB1 := false
	unixcharset := "UTF-8"
	aaplExtensions := false
	guest := "nobody"
	filemask := "DEFAULT"
	dirmask := "DEFAULT"

	_, err := r.client.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{
		NetbiosName:    &netbiosname,
		Workgroup:      &workgroup,
		Description:    &description,
		EnableSMB1:     &enableSMB1,
		UnixCharset:    &unixcharset,
		AAPLExtensions: &aaplExtensions,
		Guest:          &guest,
		Filemask:       &filemask,
		Dirmask:        &dirmask,
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

func (r *SMBConfigResource) buildUpdateRequest(plan *SMBConfigResourceModel) *client.SMBConfigUpdateRequest {
	updateReq := &client.SMBConfigUpdateRequest{}

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
	if !plan.EnableSMB1.IsNull() && !plan.EnableSMB1.IsUnknown() {
		v := plan.EnableSMB1.ValueBool()
		updateReq.EnableSMB1 = &v
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

func (r *SMBConfigResource) mapResponseToModel(config *client.SMBConfig, model *SMBConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.NetbiosName = types.StringValue(config.NetbiosName)
	model.Workgroup = types.StringValue(config.Workgroup)
	model.Description = types.StringValue(config.Description)
	model.EnableSMB1 = types.BoolValue(config.EnableSMB1)
	model.UnixCharset = types.StringValue(config.UnixCharset)
	model.AAPLExtensions = types.BoolValue(config.AAPLExtensions)
	model.Guest = types.StringValue(config.Guest)
	model.Filemask = types.StringValue(config.Filemask)
	model.Dirmask = types.StringValue(config.Dirmask)
}
