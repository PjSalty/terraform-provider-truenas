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
	_ resource.Resource                = &KeychainCredentialResource{}
	_ resource.ResourceWithImportState = &KeychainCredentialResource{}
)

// KeychainCredentialResource manages a TrueNAS keychain credential (SSH keypairs, etc).
type KeychainCredentialResource struct {
	client *client.Client
}

// KeychainCredentialResourceModel describes the resource data model.
type KeychainCredentialResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Name       types.String   `tfsdk:"name"`
	Type       types.String   `tfsdk:"type"`
	Attributes types.Map      `tfsdk:"attributes"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func NewKeychainCredentialResource() resource.Resource {
	return &KeychainCredentialResource{}
}

func (r *KeychainCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keychain_credential"
}

func (r *KeychainCredentialResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a keychain credential (SSH key pair, SSH credentials) on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the keychain credential.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the keychain credential.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"type": schema.StringAttribute{
				Description: "The credential type: SSH_KEY_PAIR or SSH_CREDENTIALS.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("SSH_KEY_PAIR", "SSH_CREDENTIALS"),
				},
			},
			"attributes": schema.MapAttribute{
				Description: "The credential attributes. For SSH_KEY_PAIR: private_key, public_key. " +
					"For SSH_CREDENTIALS: host, username, private_key, connect_timeout.",
				Required:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *KeychainCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KeychainCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create KeychainCredential start")

	var plan KeychainCredentialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert attributes from TF map to Go map
	attrs := make(map[string]interface{})
	if !plan.Attributes.IsNull() {
		var tfAttrs map[string]string
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &tfAttrs, false)...)
		for k, v := range tfAttrs {
			attrs[k] = v
		}
	}

	createReq := &client.KeychainCredentialCreateRequest{
		Name:       plan.Name.ValueString(),
		Type:       plan.Type.ValueString(),
		Attributes: attrs,
	}

	tflog.Debug(ctx, "Creating keychain credential", map[string]interface{}{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	cred, err := r.client.CreateKeychainCredential(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Keychain Credential",
			fmt.Sprintf("Could not create keychain credential %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, cred, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create KeychainCredential success")
}

func (r *KeychainCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read KeychainCredential start")

	var state KeychainCredentialResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keychain credential ID: %s", err))
		return
	}

	cred, err := r.client.GetKeychainCredential(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Keychain Credential",
			fmt.Sprintf("Could not read keychain credential %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, cred, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read KeychainCredential success")
}

func (r *KeychainCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update KeychainCredential start")

	var plan KeychainCredentialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state KeychainCredentialResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keychain credential ID: %s", err))
		return
	}

	// Convert attributes from TF map to Go map
	attrs := make(map[string]interface{})
	if !plan.Attributes.IsNull() {
		var tfAttrs map[string]string
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &tfAttrs, false)...)
		for k, v := range tfAttrs {
			attrs[k] = v
		}
	}

	updateReq := &client.KeychainCredentialUpdateRequest{
		Name:       plan.Name.ValueString(),
		Attributes: attrs,
	}

	cred, err := r.client.UpdateKeychainCredential(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Keychain Credential",
			fmt.Sprintf("Could not update keychain credential %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, cred, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update KeychainCredential success")
}

func (r *KeychainCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete KeychainCredential start")

	var state KeychainCredentialResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keychain credential ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting keychain credential", map[string]interface{}{"id": id})

	err = r.client.DeleteKeychainCredential(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Keychain credential already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Keychain Credential",
			fmt.Sprintf("Could not delete keychain credential %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete KeychainCredential success")
}

func (r *KeychainCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Keychain credential ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *KeychainCredentialResource) mapResponseToModel(ctx context.Context, cred *client.KeychainCredential, model *KeychainCredentialResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(cred.ID))
	model.Name = types.StringValue(cred.Name)
	model.Type = types.StringValue(cred.Type)

	// Convert attributes from API (map[string]interface{}) to TF map
	attrMap := make(map[string]string)
	for k, v := range cred.Attributes {
		if v != nil {
			attrMap[k] = fmt.Sprintf("%v", v)
		}
	}
	mapVal, _ := types.MapValueFrom(ctx, types.StringType, attrMap)
	model.Attributes = mapVal
}
