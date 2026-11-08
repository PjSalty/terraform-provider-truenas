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

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                   = &KeychainCredentialResource{}
	_ resource.ResourceWithImportState    = &KeychainCredentialResource{}
	_ resource.ResourceWithModifyPlan     = &KeychainCredentialResource{}
	_ resource.ResourceWithValidateConfig = &KeychainCredentialResource{}
)

// KeychainCredentialResource manages a TrueNAS keychain credential (SSH keypairs, etc).
type KeychainCredentialResource struct {
	client *wsclient.Client
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
					"For SSH_CREDENTIALS: host, port, username, private_key (the ID of an SSH_KEY_PAIR credential), " +
					"remote_host_key, connect_timeout. Values are strings; port, private_key and " +
					"connect_timeout must be whole numbers.",
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

func (r *KeychainCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create KeychainCredential start")

	var plan KeychainCredentialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	// Convert attributes from the TF string map to the wire types.
	var tfAttrs map[string]string
	if !plan.Attributes.IsNull() {
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &tfAttrs, false)...)
	}
	attrs, err := keychainWireAttributes(plan.Type.ValueString(), tfAttrs)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("attributes"), "Invalid Attributes", err.Error())
		return
	}

	createReq := &truenas.KeychainCredentialCreateRequest{
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

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keychain credential ID: %s", err))
		return
	}

	cred, err := r.client.GetKeychainCredential(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
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

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

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

	// Convert attributes from the TF string map to the wire types.
	var tfAttrs map[string]string
	if !plan.Attributes.IsNull() {
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &tfAttrs, false)...)
	}
	attrs, err := keychainWireAttributes(plan.Type.ValueString(), tfAttrs)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("attributes"), "Invalid Attributes", err.Error())
		return
	}

	updateReq := &truenas.KeychainCredentialUpdateRequest{
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

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keychain credential ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting keychain credential", map[string]interface{}{"id": id})

	err = r.client.DeleteKeychainCredential(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
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

// ModifyPlan emits a plan-time Warning whenever the plan would destroy
// this resource. Removing a keychain credential breaks any replication,
// cloud sync, or rsync task that references it.
func (r *KeychainCredentialResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_keychain_credential")
}

func (r *KeychainCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Keychain credential ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *KeychainCredentialResource) mapResponseToModel(ctx context.Context, cred *truenas.KeychainCredential, model *KeychainCredentialResourceModel) {
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

// keychainSSHCredentialsIntAttrs are the SSH_CREDENTIALS attributes the API
// types as integers. Middleware validates strictly, so the string "22" is
// rejected for port; the map schema keeps every value a string, and these are
// converted on the way out.
var keychainSSHCredentialsIntAttrs = map[string]bool{
	"port":            true,
	"private_key":     true,
	"connect_timeout": true,
}

// keychainWireAttributes converts the configured string attributes to the
// types the API expects for credType. The error names the key only: the
// attributes are sensitive, so a value is never echoed.
func keychainWireAttributes(credType string, tfAttrs map[string]string) (map[string]interface{}, error) {
	attrs := make(map[string]interface{}, len(tfAttrs))
	for k, v := range tfAttrs {
		if credType == "SSH_CREDENTIALS" && keychainSSHCredentialsIntAttrs[k] {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("attributes.%s must be a whole number for SSH_CREDENTIALS", k)
			}
			attrs[k] = n
			continue
		}
		attrs[k] = v
	}
	return attrs, nil
}

// ValidateConfig rejects, at plan time, a known SSH_CREDENTIALS integer
// attribute that is not a whole number. An unknown value (private_key wired
// from another credential's id) is left for apply.
func (r *KeychainCredentialResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg KeychainCredentialResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if cfg.Type.IsUnknown() || cfg.Type.ValueString() != "SSH_CREDENTIALS" ||
		cfg.Attributes.IsNull() || cfg.Attributes.IsUnknown() {
		return
	}
	for k, v := range cfg.Attributes.Elements() {
		sv, ok := v.(types.String)
		if !ok || !keychainSSHCredentialsIntAttrs[k] || sv.IsNull() || sv.IsUnknown() {
			continue
		}
		if _, err := strconv.Atoi(sv.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("attributes").AtMapKey(k), "Invalid Attributes",
				fmt.Sprintf("attributes.%s must be a whole number for SSH_CREDENTIALS", k))
		}
	}
}
