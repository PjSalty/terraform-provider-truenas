package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &NFSConfigResource{}
	_ resource.ResourceWithImportState = &NFSConfigResource{}
)

// NFSConfigResource manages the TrueNAS NFS service configuration.
type NFSConfigResource struct {
	client *client.Client
}

// NFSConfigResourceModel describes the resource data model.
type NFSConfigResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Servers      types.Int64    `tfsdk:"servers"`
	AllowNonroot types.Bool     `tfsdk:"allow_nonroot"`
	Protocols    types.List     `tfsdk:"protocols"`
	V4Krb        types.Bool     `tfsdk:"v4_krb"`
	V4Domain     types.String   `tfsdk:"v4_domain"`
	BindIP       types.List     `tfsdk:"bindip"`
	MountdPort   types.Int64    `tfsdk:"mountd_port"`
	RpcstatdPort types.Int64    `tfsdk:"rpcstatd_port"`
	RpclockdPort types.Int64    `tfsdk:"rpclockd_port"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func NewNFSConfigResource() resource.Resource {
	return &NFSConfigResource{}
}

func (r *NFSConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nfs_config"
}

func (r *NFSConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the NFS service configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"servers": schema.Int64Attribute{
				Description: "Number of NFS server instances (1-256).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.Between(1, 256),
				},
			},
			"allow_nonroot": schema.BoolAttribute{
				Description: "Allow non-root mount requests.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"protocols": schema.ListAttribute{
				Description: "NFS protocols to enable (e.g., NFSV3, NFSV4).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("NFSV3"),
					types.StringValue("NFSV4"),
				})),
			},
			"v4_krb": schema.BoolAttribute{
				Description: "Enable NFSv4 Kerberos support.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"v4_domain": schema.StringAttribute{
				Description: "NFSv4 ID mapping domain.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
			},
			"bindip": schema.ListAttribute{
				Description: "IP addresses to bind the NFS service to. Empty means all addresses.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"mountd_port": schema.Int64Attribute{
				Description: "Port for mountd. 0 means random.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"rpcstatd_port": schema.Int64Attribute{
				Description: "Port for rpc.statd. 0 means random.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"rpclockd_port": schema.Int64Attribute{
				Description: "Port for rpc.lockd. 0 means random.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NFSConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NFSConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NFSConfig start")

	var plan NFSConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating NFS config resource (updating singleton)")

	var d diag.Diagnostics
	updateReq := r.buildUpdateRequest(ctx, &plan, &d)
	resp.Diagnostics.Append(d...)

	config, err := r.client.UpdateNFSConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NFS Config",
			fmt.Sprintf("Could not update NFS configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NFSConfig success")
}

func (r *NFSConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NFSConfig start")

	var state NFSConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetNFSConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NFS Config",
			fmt.Sprintf("Could not read NFS configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NFSConfig success")
}

func (r *NFSConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NFSConfig start")

	var plan NFSConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var d diag.Diagnostics
	updateReq := r.buildUpdateRequest(ctx, &plan, &d)
	resp.Diagnostics.Append(d...)

	config, err := r.client.UpdateNFSConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NFS Config",
			fmt.Sprintf("Could not update NFS configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NFSConfig success")
}

func (r *NFSConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NFSConfig start")

	tflog.Debug(ctx, "Deleting NFS config resource (resetting to defaults)")

	// Reset to defaults — singleton cannot be deleted
	servers := 2
	allowNonroot := false
	v4Krb := false
	v4Domain := ""

	_, err := r.client.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{
		Servers:      &servers,
		AllowNonroot: &allowNonroot,
		Protocols:    []string{"NFSV3", "NFSV4"},
		V4Krb:        &v4Krb,
		V4Domain:     &v4Domain,
		BindIP:       []string{},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting NFS Config",
			fmt.Sprintf("Could not reset NFS configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NFSConfig success")
}

func (r *NFSConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NFSConfigResource) buildUpdateRequest(ctx context.Context, plan *NFSConfigResourceModel, d *diag.Diagnostics) *client.NFSConfigUpdateRequest {
	updateReq := &client.NFSConfigUpdateRequest{}

	if !plan.Servers.IsNull() && !plan.Servers.IsUnknown() {
		v := int(plan.Servers.ValueInt64())
		updateReq.Servers = &v
	}
	if !plan.AllowNonroot.IsNull() && !plan.AllowNonroot.IsUnknown() {
		v := plan.AllowNonroot.ValueBool()
		updateReq.AllowNonroot = &v
	}
	if !plan.V4Krb.IsNull() && !plan.V4Krb.IsUnknown() {
		v := plan.V4Krb.ValueBool()
		updateReq.V4Krb = &v
	}
	if !plan.V4Domain.IsNull() && !plan.V4Domain.IsUnknown() {
		v := plan.V4Domain.ValueString()
		updateReq.V4Domain = &v
	}

	if !plan.Protocols.IsNull() && !plan.Protocols.IsUnknown() {
		var protocols []string
		diags := plan.Protocols.ElementsAs(ctx, &protocols, false)
		d.Append(diags...)
		updateReq.Protocols = protocols
	}

	if !plan.BindIP.IsNull() && !plan.BindIP.IsUnknown() {
		var bindip []string
		diags := plan.BindIP.ElementsAs(ctx, &bindip, false)
		d.Append(diags...)
		updateReq.BindIP = bindip
	}

	if !plan.MountdPort.IsNull() && !plan.MountdPort.IsUnknown() {
		v := int(plan.MountdPort.ValueInt64())
		updateReq.MountdPort = &v
	}
	if !plan.RpcstatdPort.IsNull() && !plan.RpcstatdPort.IsUnknown() {
		v := int(plan.RpcstatdPort.ValueInt64())
		updateReq.RpcstatdPort = &v
	}
	if !plan.RpclockdPort.IsNull() && !plan.RpclockdPort.IsUnknown() {
		v := int(plan.RpclockdPort.ValueInt64())
		updateReq.RpclockdPort = &v
	}

	return updateReq
}

func (r *NFSConfigResource) mapResponseToModel(ctx context.Context, config *client.NFSConfig, model *NFSConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.Servers = types.Int64Value(int64(config.Servers))
	model.AllowNonroot = types.BoolValue(config.AllowNonroot)
	model.V4Krb = types.BoolValue(config.V4Krb)
	model.V4Domain = types.StringValue(config.V4Domain)

	protocolValues, diags := types.ListValueFrom(ctx, types.StringType, config.Protocols)
	if !diags.HasError() {
		model.Protocols = protocolValues
	}

	bindipValues, diags := types.ListValueFrom(ctx, types.StringType, config.BindIP)
	if !diags.HasError() {
		model.BindIP = bindipValues
	}

	if config.MountdPort != nil {
		model.MountdPort = types.Int64Value(int64(*config.MountdPort))
	} else {
		model.MountdPort = types.Int64Null()
	}
	if config.RpcstatdPort != nil {
		model.RpcstatdPort = types.Int64Value(int64(*config.RpcstatdPort))
	} else {
		model.RpcstatdPort = types.Int64Null()
	}
	if config.RpclockdPort != nil {
		model.RpclockdPort = types.Int64Value(int64(*config.RpclockdPort))
	} else {
		model.RpclockdPort = types.Int64Null()
	}
}
