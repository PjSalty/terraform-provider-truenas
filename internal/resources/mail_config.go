package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	_ resource.Resource                = &MailConfigResource{}
	_ resource.ResourceWithImportState = &MailConfigResource{}
)

// MailConfigResource manages the TrueNAS mail/SMTP configuration.
type MailConfigResource struct {
	client *client.Client
}

// MailConfigResourceModel describes the resource data model.
type MailConfigResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	FromEmail      types.String   `tfsdk:"fromemail"`
	FromName       types.String   `tfsdk:"fromname"`
	OutgoingServer types.String   `tfsdk:"outgoingserver"`
	Port           types.Int64    `tfsdk:"port"`
	Security       types.String   `tfsdk:"security"`
	SMTP           types.Bool     `tfsdk:"smtp"`
	User           types.String   `tfsdk:"user"`
	Pass           types.String   `tfsdk:"pass"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func NewMailConfigResource() resource.Resource {
	return &MailConfigResource{}
}

func (r *MailConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mail_config"
}

func (r *MailConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the email/SMTP configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fromemail": schema.StringAttribute{
				Description: "From email address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fromname": schema.StringAttribute{
				Description: "From name.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 120),
				},
			},
			"outgoingserver": schema.StringAttribute{
				Description: "Outgoing SMTP server hostname or IP.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
			},
			"port": schema.Int64Attribute{
				Description: "SMTP port.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(25),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"security": schema.StringAttribute{
				Description: "Email security setting (PLAIN, SSL, TLS).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("PLAIN"),
				Validators: []validator.String{
					stringvalidator.OneOf("PLAIN", "SSL", "TLS"),
				},
			},
			"smtp": schema.BoolAttribute{
				Description: "Enable SMTP authentication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"user": schema.StringAttribute{
				Description: "SMTP authentication username.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"pass": schema.StringAttribute{
				Description: "SMTP authentication password.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
		},
	}
}

func (r *MailConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MailConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create MailConfig start")

	var plan MailConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating mail config resource (updating singleton)")

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateMailConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Mail Config",
			fmt.Sprintf("Could not update mail configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create MailConfig success")
}

func (r *MailConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read MailConfig start")

	var state MailConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetMailConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Mail Config",
			fmt.Sprintf("Could not read mail configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read MailConfig success")
}

func (r *MailConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update MailConfig start")

	var plan MailConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateMailConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Mail Config",
			fmt.Sprintf("Could not update mail configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update MailConfig success")
}

func (r *MailConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete MailConfig start")

	tflog.Debug(ctx, "Deleting mail config resource (resetting to defaults)")

	fromname := ""
	outgoingserver := ""
	port := 25
	security := "PLAIN"
	smtp := false
	user := ""
	pass := ""

	// fromemail is not reset because the API rejects empty values
	_, err := r.client.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{
		FromName:       &fromname,
		OutgoingServer: &outgoingserver,
		Port:           &port,
		Security:       &security,
		SMTP:           &smtp,
		User:           &user,
		Pass:           &pass,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting Mail Config",
			fmt.Sprintf("Could not reset mail configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete MailConfig success")
}

func (r *MailConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *MailConfigResource) buildUpdateRequest(plan *MailConfigResourceModel) *client.MailConfigUpdateRequest {
	updateReq := &client.MailConfigUpdateRequest{}

	if !plan.FromEmail.IsNull() && !plan.FromEmail.IsUnknown() {
		v := plan.FromEmail.ValueString()
		if v != "" {
			updateReq.FromEmail = &v
		}
	}
	if !plan.FromName.IsNull() && !plan.FromName.IsUnknown() {
		v := plan.FromName.ValueString()
		updateReq.FromName = &v
	}
	if !plan.OutgoingServer.IsNull() && !plan.OutgoingServer.IsUnknown() {
		v := plan.OutgoingServer.ValueString()
		updateReq.OutgoingServer = &v
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		v := int(plan.Port.ValueInt64())
		updateReq.Port = &v
	}
	if !plan.Security.IsNull() && !plan.Security.IsUnknown() {
		v := plan.Security.ValueString()
		updateReq.Security = &v
	}
	if !plan.SMTP.IsNull() && !plan.SMTP.IsUnknown() {
		v := plan.SMTP.ValueBool()
		updateReq.SMTP = &v
	}
	if !plan.User.IsNull() && !plan.User.IsUnknown() {
		v := plan.User.ValueString()
		updateReq.User = &v
	}
	if !plan.Pass.IsNull() && !plan.Pass.IsUnknown() {
		v := plan.Pass.ValueString()
		updateReq.Pass = &v
	}

	return updateReq
}

func (r *MailConfigResource) mapResponseToModel(config *client.MailConfig, model *MailConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.FromEmail = types.StringValue(config.FromEmail)
	model.FromName = types.StringValue(config.FromName)
	model.OutgoingServer = types.StringValue(config.OutgoingServer)
	model.Port = types.Int64Value(int64(config.Port))
	model.Security = types.StringValue(config.Security)
	model.SMTP = types.BoolValue(config.SMTP)
	if config.User != nil {
		model.User = types.StringValue(*config.User)
	} else {
		model.User = types.StringValue("")
	}
	model.Pass = types.StringValue(config.Pass)
}
