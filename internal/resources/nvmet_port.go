package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &NVMetPortResource{}
	_ resource.ResourceWithImportState = &NVMetPortResource{}
)

// NVMetPortResource manages an NVMe-oF transport port.
type NVMetPortResource struct {
	client *client.Client
}

// NVMetPortResourceModel describes the resource data model.
type NVMetPortResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	Index          types.Int64    `tfsdk:"index"`
	AddrTrtype     types.String   `tfsdk:"addr_trtype"`
	AddrTraddr     types.String   `tfsdk:"addr_traddr"`
	AddrTrsvcid    types.Int64    `tfsdk:"addr_trsvcid"`
	InlineDataSize types.Int64    `tfsdk:"inline_data_size"`
	MaxQueueSize   types.Int64    `tfsdk:"max_queue_size"`
	PiEnable       types.Bool     `tfsdk:"pi_enable"`
	Enabled        types.Bool     `tfsdk:"enabled"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetPortResource() resource.Resource {
	return &NVMetPortResource{}
}

func (r *NVMetPortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_port"
}

func (r *NVMetPortResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NVMe-oF transport port on TrueNAS SCALE.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the NVMe-oF port.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"index": schema.Int64Attribute{
				Description: "Internal port index.",
				Computed:    true,
			},
			"addr_trtype": schema.StringAttribute{
				Description: "Fabric transport technology: TCP, RDMA, or FC.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("TCP", "RDMA", "FC"),
				},
			},
			"addr_traddr": schema.StringAttribute{
				Description: "Transport address. For TCP/RDMA, an IPv4/IPv6 address. For FC, a fabric-specific address.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 253),
				},
			},
			"addr_trsvcid": schema.Int64Attribute{
				Description: "Transport service ID. For TCP/RDMA, the port number (default 4420). Not used for FC.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"inline_data_size": schema.Int64Attribute{
				Description: "Maximum size for inline data transfers in bytes (0-1048576).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 1048576),
				},
			},
			"max_queue_size": schema.Int64Attribute{
				Description: "Maximum number of queue entries (1-65535).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"pi_enable": schema.BoolAttribute{
				Description: "Whether Protection Information (PI) is enabled.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the port is enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NVMetPortResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetPort start")

	var plan NVMetPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NVMetPortCreateRequest{
		AddrTrtype: plan.AddrTrtype.ValueString(),
		AddrTraddr: plan.AddrTraddr.ValueString(),
	}
	if !plan.AddrTrsvcid.IsNull() && !plan.AddrTrsvcid.IsUnknown() {
		v := int(plan.AddrTrsvcid.ValueInt64())
		createReq.AddrTrsvcid = &v
	}
	if !plan.InlineDataSize.IsNull() && !plan.InlineDataSize.IsUnknown() {
		v := int(plan.InlineDataSize.ValueInt64())
		createReq.InlineDataSize = &v
	}
	if !plan.MaxQueueSize.IsNull() && !plan.MaxQueueSize.IsUnknown() {
		v := int(plan.MaxQueueSize.ValueInt64())
		createReq.MaxQueueSize = &v
	}
	if !plan.PiEnable.IsNull() && !plan.PiEnable.IsUnknown() {
		v := plan.PiEnable.ValueBool()
		createReq.PiEnable = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		createReq.Enabled = &v
	}

	tflog.Debug(ctx, "Creating nvmet_port", map[string]interface{}{
		"trtype": createReq.AddrTrtype,
		"traddr": createReq.AddrTraddr,
	})

	port, err := r.client.CreateNVMetPort(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NVMe-oF Port",
			fmt.Sprintf("Could not create NVMe-oF port: %s", err),
		)
		return
	}

	r.mapResponseToModel(port, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetPort success")
}

func (r *NVMetPortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetPort start")

	var state NVMetPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF port ID: %s", err))
		return
	}

	port, err := r.client.GetNVMetPort(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Port",
			fmt.Sprintf("Could not read NVMe-oF port %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(port, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetPort success")
}

func (r *NVMetPortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetPort start")

	var plan NVMetPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NVMetPortResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF port ID: %s", err))
		return
	}

	updateReq := &client.NVMetPortUpdateRequest{}
	if !plan.AddrTrtype.IsNull() && !plan.AddrTrtype.IsUnknown() {
		v := plan.AddrTrtype.ValueString()
		updateReq.AddrTrtype = &v
	}
	if !plan.AddrTraddr.IsNull() && !plan.AddrTraddr.IsUnknown() {
		v := plan.AddrTraddr.ValueString()
		updateReq.AddrTraddr = &v
	}
	if !plan.AddrTrsvcid.IsNull() && !plan.AddrTrsvcid.IsUnknown() {
		v := int(plan.AddrTrsvcid.ValueInt64())
		updateReq.AddrTrsvcid = &v
	}
	if !plan.InlineDataSize.IsNull() && !plan.InlineDataSize.IsUnknown() {
		v := int(plan.InlineDataSize.ValueInt64())
		updateReq.InlineDataSize = &v
	}
	if !plan.MaxQueueSize.IsNull() && !plan.MaxQueueSize.IsUnknown() {
		v := int(plan.MaxQueueSize.ValueInt64())
		updateReq.MaxQueueSize = &v
	}
	if !plan.PiEnable.IsNull() && !plan.PiEnable.IsUnknown() {
		v := plan.PiEnable.ValueBool()
		updateReq.PiEnable = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		updateReq.Enabled = &v
	}

	port, err := r.client.UpdateNVMetPort(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NVMe-oF Port",
			fmt.Sprintf("Could not update NVMe-oF port %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(port, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetPort success")
}

func (r *NVMetPortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetPort start")

	var state NVMetPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF port ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting nvmet_port", map[string]interface{}{"id": id})

	if err := r.client.DeleteNVMetPort(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NVMe-oF port already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NVMe-oF Port",
			fmt.Sprintf("Could not delete NVMe-oF port %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NVMetPort success")
}

func (r *NVMetPortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NVMe-oF port ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetPortResource) mapResponseToModel(port *client.NVMetPort, model *NVMetPortResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(port.ID))
	model.Index = types.Int64Value(int64(port.Index))
	model.AddrTrtype = types.StringValue(port.AddrTrtype)
	model.AddrTraddr = types.StringValue(port.AddrTraddr)
	svcid := port.GetAddrTrsvcid()
	if svcid != 0 {
		model.AddrTrsvcid = types.Int64Value(int64(svcid))
	} else {
		model.AddrTrsvcid = types.Int64Null()
	}
	if port.InlineDataSize != nil {
		model.InlineDataSize = types.Int64Value(int64(*port.InlineDataSize))
	} else {
		model.InlineDataSize = types.Int64Null()
	}
	if port.MaxQueueSize != nil {
		model.MaxQueueSize = types.Int64Value(int64(*port.MaxQueueSize))
	} else {
		model.MaxQueueSize = types.Int64Null()
	}
	if port.PiEnable != nil {
		model.PiEnable = types.BoolValue(*port.PiEnable)
	} else {
		model.PiEnable = types.BoolNull()
	}
	model.Enabled = types.BoolValue(port.Enabled)
}
