package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &APIKeyResource{}
	_ resource.ResourceWithImportState = &APIKeyResource{}
)

// APIKeyResource manages a TrueNAS API key.
type APIKeyResource struct {
	client *client.Client
}

// APIKeyResourceModel describes the resource data model.
type APIKeyResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Username types.String   `tfsdk:"username"`
	Key      types.String   `tfsdk:"key"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an API key on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the API key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the API key.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"username": schema.StringAttribute{
				Description: "The username this API key is associated with.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The API key value. Only available on creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create APIKey start")

	var plan APIKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.APIKeyCreateRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Username.IsNull() {
		createReq.Username = plan.Username.ValueString()
	}

	tflog.Debug(ctx, "Creating API key", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	apiKey, err := r.client.CreateAPIKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Key",
			fmt.Sprintf("Could not create API key %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(apiKey.ID))
	plan.Name = types.StringValue(apiKey.Name)
	plan.Username = types.StringValue(apiKey.Username)
	// Key is only returned on creation
	plan.Key = types.StringValue(apiKey.Key)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create APIKey success")
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read APIKey start")

	var state APIKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse API key ID: %s", err))
		return
	}

	apiKey, err := r.client.GetAPIKey(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading API Key",
			fmt.Sprintf("Could not read API key %d: %s", id, err),
		)
		return
	}

	// Preserve the key from state since it is only returned on creation
	key := state.Key
	state.ID = types.StringValue(strconv.Itoa(apiKey.ID))
	state.Name = types.StringValue(apiKey.Name)
	state.Username = types.StringValue(apiKey.Username)
	state.Key = key

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read APIKey success")
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update APIKey start")

	var plan APIKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state APIKeyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse API key ID: %s", err))
		return
	}

	updateReq := &client.APIKeyUpdateRequest{
		Name: plan.Name.ValueString(),
	}

	apiKey, err := r.client.UpdateAPIKey(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating API Key",
			fmt.Sprintf("Could not update API key %d: %s", id, err),
		)
		return
	}

	// Preserve the key from state since it is only returned on creation
	plan.ID = types.StringValue(strconv.Itoa(apiKey.ID))
	plan.Name = types.StringValue(apiKey.Name)
	plan.Username = types.StringValue(apiKey.Username)
	plan.Key = state.Key

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update APIKey success")
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete APIKey start")

	var state APIKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse API key ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting API key", map[string]interface{}{"id": id})

	err = r.client.DeleteAPIKey(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "API key already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting API Key",
			fmt.Sprintf("Could not delete API key %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete APIKey success")
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("API key ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), types.StringValue(""))...)
}
