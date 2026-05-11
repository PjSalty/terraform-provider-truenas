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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &ACMEDNSAuthenticatorResource{}
	_ resource.ResourceWithImportState = &ACMEDNSAuthenticatorResource{}
)

// ACMEDNSAuthenticatorResource manages a TrueNAS ACME DNS authenticator.
type ACMEDNSAuthenticatorResource struct {
	client *client.Client
}

// ACMEDNSAuthenticatorResourceModel describes the resource data model.
type ACMEDNSAuthenticatorResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Name          types.String   `tfsdk:"name"`
	Authenticator types.String   `tfsdk:"authenticator"`
	Attributes    types.Map      `tfsdk:"attributes"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewACMEDNSAuthenticatorResource() resource.Resource {
	return &ACMEDNSAuthenticatorResource{}
}

func (r *ACMEDNSAuthenticatorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acme_dns_authenticator"
}

func (r *ACMEDNSAuthenticatorResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an ACME DNS authenticator on TrueNAS SCALE for Let's Encrypt certificate validation." + "\n\n" +
		"**Stability: Alpha.** Not end-to-end verified — acceptance tests require a real DNS provider API token (Cloudflare, Route53, DigitalOcean). Schema and wire format match the TrueNAS REST API but no live create/read/destroy cycle has been observed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the ACME DNS authenticator.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the ACME DNS authenticator.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"authenticator": schema.StringAttribute{
				Description: "The DNS provider type (e.g., cloudflare, route53, digitalocean).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"attributes": schema.MapAttribute{
				Description: "Provider-specific configuration attributes (e.g., api_token for Cloudflare).",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ACMEDNSAuthenticatorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ACMEDNSAuthenticatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ACMEDNSAuthenticator start")

	var plan ACMEDNSAuthenticatorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build attributes map — the authenticator type goes inside attributes
	attrs := map[string]interface{}{
		"authenticator": plan.Authenticator.ValueString(),
	}
	if !plan.Attributes.IsNull() {
		var tfAttrs map[string]string
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &tfAttrs, false)...)
		for k, v := range tfAttrs {
			attrs[k] = v
		}
	}

	createReq := &client.ACMEDNSAuthenticatorCreateRequest{
		Name:       plan.Name.ValueString(),
		Attributes: attrs,
	}

	tflog.Debug(ctx, "Creating ACME DNS authenticator", map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"authenticator": plan.Authenticator.ValueString(),
	})

	auth, err := r.client.CreateACMEDNSAuthenticator(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ACME DNS Authenticator",
			fmt.Sprintf("Could not create ACME DNS authenticator %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, auth, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ACMEDNSAuthenticator success")
}

func (r *ACMEDNSAuthenticatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ACMEDNSAuthenticator start")

	var state ACMEDNSAuthenticatorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse ACME DNS authenticator ID: %s", err))
		return
	}

	auth, err := r.client.GetACMEDNSAuthenticator(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading ACME DNS Authenticator",
			fmt.Sprintf("Could not read ACME DNS authenticator %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, auth, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ACMEDNSAuthenticator success")
}

func (r *ACMEDNSAuthenticatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ACMEDNSAuthenticator start")

	var plan ACMEDNSAuthenticatorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ACMEDNSAuthenticatorResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse ACME DNS authenticator ID: %s", err))
		return
	}

	// Build attributes map
	attrs := map[string]interface{}{
		"authenticator": plan.Authenticator.ValueString(),
	}
	if !plan.Attributes.IsNull() {
		var tfAttrs map[string]string
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &tfAttrs, false)...)
		for k, v := range tfAttrs {
			attrs[k] = v
		}
	}

	updateReq := &client.ACMEDNSAuthenticatorUpdateRequest{
		Name:       plan.Name.ValueString(),
		Attributes: attrs,
	}

	auth, err := r.client.UpdateACMEDNSAuthenticator(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating ACME DNS Authenticator",
			fmt.Sprintf("Could not update ACME DNS authenticator %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, auth, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ACMEDNSAuthenticator success")
}

func (r *ACMEDNSAuthenticatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ACMEDNSAuthenticator start")

	var state ACMEDNSAuthenticatorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse ACME DNS authenticator ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting ACME DNS authenticator", map[string]interface{}{"id": id})

	err = r.client.DeleteACMEDNSAuthenticator(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "ACME DNS authenticator already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting ACME DNS Authenticator",
			fmt.Sprintf("Could not delete ACME DNS authenticator %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ACMEDNSAuthenticator success")
}

func (r *ACMEDNSAuthenticatorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("ACME DNS authenticator ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ACMEDNSAuthenticatorResource) mapResponseToModel(ctx context.Context, auth *client.ACMEDNSAuthenticator, model *ACMEDNSAuthenticatorResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(auth.ID))
	model.Name = types.StringValue(auth.Name)

	// Extract authenticator from attributes
	attrMap := make(map[string]string)
	for k, v := range auth.Attributes {
		if k == "authenticator" {
			if s, ok := v.(string); ok {
				model.Authenticator = types.StringValue(s)
			}
			continue
		}
		if v != nil {
			attrMap[k] = fmt.Sprintf("%v", v)
		}
	}

	mapVal, _ := types.MapValueFrom(ctx, types.StringType, attrMap)
	model.Attributes = mapVal
}
