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
	_ resource.Resource                = &SNMPConfigResource{}
	_ resource.ResourceWithImportState = &SNMPConfigResource{}
)

// SNMPConfigResource manages the TrueNAS SNMP configuration.
type SNMPConfigResource struct {
	client *client.Client
}

// SNMPConfigResourceModel describes the resource data model.
type SNMPConfigResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	Community        types.String   `tfsdk:"community"`
	Contact          types.String   `tfsdk:"contact"`
	Location         types.String   `tfsdk:"location"`
	V3               types.Bool     `tfsdk:"v3"`
	V3Username       types.String   `tfsdk:"v3_username"`
	V3AuthType       types.String   `tfsdk:"v3_authtype"`
	V3Password       types.String   `tfsdk:"v3_password"`
	V3PrivProto      types.String   `tfsdk:"v3_privproto"`
	V3PrivPassphrase types.String   `tfsdk:"v3_privpassphrase"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

func NewSNMPConfigResource() resource.Resource {
	return &SNMPConfigResource{}
}

func (r *SNMPConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snmp_config"
}

func (r *SNMPConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the SNMP configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"community": schema.StringAttribute{
				Description: "SNMP community string.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("public"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 120),
				},
			},
			"contact": schema.StringAttribute{
				Description: "SNMP contact information.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			"location": schema.StringAttribute{
				Description: "SNMP system location.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			"v3": schema.BoolAttribute{
				Description: "Enable SNMPv3 support.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"v3_username": schema.StringAttribute{
				Description: "SNMPv3 username.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(32),
				},
			},
			"v3_authtype": schema.StringAttribute{
				Description: "SNMPv3 authentication type (SHA, MD5).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SHA"),
				Validators: []validator.String{
					stringvalidator.OneOf("SHA", "MD5", ""),
				},
			},
			"v3_password": schema.StringAttribute{
				Description: "SNMPv3 authentication password.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.Any(
						stringvalidator.LengthBetween(0, 0),
						stringvalidator.LengthAtLeast(8),
					),
				},
			},
			"v3_privproto": schema.StringAttribute{
				Description: "SNMPv3 privacy protocol (AES, DES).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("AES", "DES", ""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"v3_privpassphrase": schema.StringAttribute{
				Description: "SNMPv3 privacy passphrase (8+ chars or empty).",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.Any(
						stringvalidator.LengthBetween(0, 0),
						stringvalidator.LengthAtLeast(8),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SNMPConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SNMPConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SNMPConfig start")

	var plan SNMPConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SNMP config resource (updating singleton)")

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateSNMPConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SNMP Config",
			fmt.Sprintf("Could not update SNMP configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SNMPConfig success")
}

func (r *SNMPConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SNMPConfig start")

	var state SNMPConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetSNMPConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SNMP Config",
			fmt.Sprintf("Could not read SNMP configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SNMPConfig success")
}

func (r *SNMPConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SNMPConfig start")

	var plan SNMPConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateSNMPConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SNMP Config",
			fmt.Sprintf("Could not update SNMP configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SNMPConfig success")
}

func (r *SNMPConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SNMPConfig start")

	tflog.Debug(ctx, "Deleting SNMP config resource (resetting to defaults)")

	community := "public"
	contact := ""
	location := ""
	v3 := false
	v3Username := ""
	v3AuthType := "SHA"
	v3Password := ""

	_, err := r.client.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{
		Community:  &community,
		Contact:    &contact,
		Location:   &location,
		V3:         &v3,
		V3Username: &v3Username,
		V3AuthType: &v3AuthType,
		V3Password: &v3Password,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting SNMP Config",
			fmt.Sprintf("Could not reset SNMP configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete SNMPConfig success")
}

func (r *SNMPConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SNMPConfigResource) buildUpdateRequest(plan *SNMPConfigResourceModel) *client.SNMPConfigUpdateRequest {
	updateReq := &client.SNMPConfigUpdateRequest{}

	if !plan.Community.IsNull() && !plan.Community.IsUnknown() {
		v := plan.Community.ValueString()
		updateReq.Community = &v
	}
	if !plan.Contact.IsNull() && !plan.Contact.IsUnknown() {
		v := plan.Contact.ValueString()
		updateReq.Contact = &v
	}
	if !plan.Location.IsNull() && !plan.Location.IsUnknown() {
		v := plan.Location.ValueString()
		updateReq.Location = &v
	}
	if !plan.V3.IsNull() && !plan.V3.IsUnknown() {
		v := plan.V3.ValueBool()
		updateReq.V3 = &v
	}
	if !plan.V3Username.IsNull() && !plan.V3Username.IsUnknown() {
		v := plan.V3Username.ValueString()
		updateReq.V3Username = &v
	}
	if !plan.V3AuthType.IsNull() && !plan.V3AuthType.IsUnknown() {
		v := plan.V3AuthType.ValueString()
		updateReq.V3AuthType = &v
	}
	if !plan.V3Password.IsNull() && !plan.V3Password.IsUnknown() {
		v := plan.V3Password.ValueString()
		updateReq.V3Password = &v
	}
	if !plan.V3PrivProto.IsNull() && !plan.V3PrivProto.IsUnknown() {
		v := plan.V3PrivProto.ValueString()
		updateReq.V3PrivProto = &v
	}
	if !plan.V3PrivPassphrase.IsNull() && !plan.V3PrivPassphrase.IsUnknown() {
		v := plan.V3PrivPassphrase.ValueString()
		updateReq.V3PrivPassphrase = &v
	}

	return updateReq
}

func (r *SNMPConfigResource) mapResponseToModel(config *client.SNMPConfig, model *SNMPConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.Community = types.StringValue(config.Community)
	model.Contact = types.StringValue(config.Contact)
	model.Location = types.StringValue(config.Location)
	model.V3 = types.BoolValue(config.V3)
	model.V3Username = types.StringValue(config.V3Username)
	model.V3AuthType = types.StringValue(config.V3AuthType)
	model.V3Password = types.StringValue(config.V3Password)

	if config.V3PrivProto != nil {
		model.V3PrivProto = types.StringValue(*config.V3PrivProto)
	} else {
		model.V3PrivProto = types.StringNull()
	}
	if config.V3PrivPassphrase != nil {
		model.V3PrivPassphrase = types.StringValue(*config.V3PrivPassphrase)
	} else {
		model.V3PrivPassphrase = types.StringNull()
	}
}
