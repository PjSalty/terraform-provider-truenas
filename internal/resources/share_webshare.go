package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                = &WebshareResource{}
	_ resource.ResourceWithImportState = &WebshareResource{}
)

// WebshareResource manages a WebShare share, the share protocol TrueNAS
// added in 26.0. The sharing.webshare namespace does not exist on 25.10 or
// earlier, so this resource is unusable there; the client turns that into a
// diagnostic naming the required version rather than a bare
// method-not-found.
type WebshareResource struct {
	client *wsclient.Client
}

// WebshareResourceModel describes the resource data model.
//
// Only name, path, enabled and is_home_base are settable. dataset,
// relative_path and locked are excluded_field() on the upstream create
// model, so they are derived by middleware and exposed read-only. Making
// them settable would let a config express something the server silently
// ignores.
type WebshareResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	Path         types.String   `tfsdk:"path"`
	Enabled      types.Bool     `tfsdk:"enabled"`
	IsHomeBase   types.Bool     `tfsdk:"is_home_base"`
	Dataset      types.String   `tfsdk:"dataset"`
	RelativePath types.String   `tfsdk:"relative_path"`
	Locked       types.Bool     `tfsdk:"locked"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

// NewWebshareResource returns a new WebshareResource factory.
func NewWebshareResource() resource.Resource {
	return &WebshareResource{}
}

func (r *WebshareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_webshare"
}

func (r *WebshareResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages a WebShare share on TrueNAS SCALE. WebShare is the browser-based share " +
			"protocol introduced in TrueNAS 26.0; this resource cannot be used against 25.10 or older, " +
			"which have no sharing.webshare namespace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The WebShare share ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The share name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"path": schema.StringAttribute{
				Description: "Filesystem path exposed by the share.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the share is available. Defaults to true, matching the server.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"is_home_base": schema.BoolAttribute{
				Description: "Whether this share is the home base, under which per-user home shares are created.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			// Derived server-side. Excluded from the upstream create model,
			// so they are reported, never sent.
			"dataset": schema.StringAttribute{
				Description: "Dataset backing the share path. Derived by TrueNAS, read-only.",
				Computed:    true,
			},
			"relative_path": schema.StringAttribute{
				Description: "Path relative to the backing dataset. Derived by TrueNAS, read-only.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the share's dataset is locked by encryption. Derived by TrueNAS, read-only.",
				Computed:    true,
			},
		},
	}
}

func (r *WebshareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebshareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Webshare start")

	var plan WebshareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled := plan.Enabled.ValueBool()
	isHomeBase := plan.IsHomeBase.ValueBool()
	createReq := &truenas.WebshareCreateRequest{
		Name:       plan.Name.ValueString(),
		Path:       plan.Path.ValueString(),
		Enabled:    &enabled,
		IsHomeBase: &isHomeBase,
	}

	share, err := r.client.CreateWebshare(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating WebShare Share",
			fmt.Sprintf("Could not create WebShare share %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(share, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create Webshare success")
}

func (r *WebshareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Webshare start")

	var state WebshareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("WebShare share ID must be numeric: %s", err))
		return
	}

	share, err := r.client.GetWebshare(ctx, id)
	if err != nil {
		// A share deleted out of band drops from state so the next plan
		// recreates it, rather than failing every run.
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading WebShare Share",
			fmt.Sprintf("Could not read WebShare share %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(share, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read Webshare success")
}

func (r *WebshareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Webshare start")

	var plan, state WebshareResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("WebShare share ID must be numeric: %s", err))
		return
	}

	name := plan.Name.ValueString()
	pathVal := plan.Path.ValueString()
	enabled := plan.Enabled.ValueBool()
	isHomeBase := plan.IsHomeBase.ValueBool()
	updateReq := &truenas.WebshareUpdateRequest{
		Name:       &name,
		Path:       &pathVal,
		Enabled:    &enabled,
		IsHomeBase: &isHomeBase,
	}

	share, err := r.client.UpdateWebshare(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating WebShare Share",
			fmt.Sprintf("Could not update WebShare share %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(share, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update Webshare success")
}

func (r *WebshareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Webshare start")

	var state WebshareResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("WebShare share ID must be numeric: %s", err))
		return
	}

	if err := r.client.DeleteWebshare(ctx, id); err != nil {
		// Already gone is success: the desired end state is reached.
		if wsclient.IsNotFound(err) {
			tflog.Trace(ctx, "Delete Webshare: already absent")
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting WebShare Share",
			fmt.Sprintf("Could not delete WebShare share %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Webshare success")
}

func (r *WebshareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("WebShare share ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToModel writes the server's view into the model.
//
// The three derived fields are nullable upstream. They are written as
// known values rather than null so a plan does not show them as
// "(known after apply)" on every run.
func (r *WebshareResource) mapResponseToModel(share *truenas.Webshare, model *WebshareResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(share.ID))
	model.Name = types.StringValue(share.Name)
	model.Path = types.StringValue(share.Path)
	model.Enabled = types.BoolValue(share.Enabled)
	model.IsHomeBase = types.BoolValue(share.IsHomeBase)

	model.Dataset = types.StringValue("")
	if share.Dataset != nil {
		model.Dataset = types.StringValue(*share.Dataset)
	}
	model.RelativePath = types.StringValue("")
	if share.RelativePath != nil {
		model.RelativePath = types.StringValue(*share.RelativePath)
	}
	model.Locked = types.BoolValue(share.Locked != nil && *share.Locked)
}
