package resources

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// iscsiGlobalSingletonID is the fixed identifier for truenas_iscsi_global.
// TrueNAS has exactly one iSCSI global configuration, so the ID is a
// constant.
const iscsiGlobalSingletonID = "iscsi_global"

var (
	_ resource.Resource                = &ISCSIGlobalResource{}
	_ resource.ResourceWithImportState = &ISCSIGlobalResource{}
)

// ISCSIGlobalResource manages the iSCSI global configuration.
type ISCSIGlobalResource struct {
	client *wsclient.Client
}

// ISCSIGlobalResourceModel describes the resource data model.
type ISCSIGlobalResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Basename           types.String   `tfsdk:"basename"`
	IsnsServers        types.Set      `tfsdk:"isns_servers"`
	ListenPort         types.Int64    `tfsdk:"listen_port"`
	PoolAvailThreshold types.Int64    `tfsdk:"pool_avail_threshold"`
	Alua               types.Bool     `tfsdk:"alua"`
	Iser               types.Bool     `tfsdk:"iser"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

// NewISCSIGlobalResource returns a new ISCSIGlobalResource factory.
func NewISCSIGlobalResource() resource.Resource {
	return &ISCSIGlobalResource{}
}

func (r *ISCSIGlobalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_global"
}

func (r *ISCSIGlobalResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the iSCSI global configuration on TrueNAS SCALE: the target name prefix, " +
			"the iSNS servers, the listen port and the pool space warning. This is a singleton, " +
			"TrueNAS has exactly one.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Fixed singleton identifier. Always \"iscsi_global\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"basename": schema.StringAttribute{
				Description: "Prefix of every target IQN (`<basename>:<target name>`). Left out, the " +
					"current value is kept; TrueNAS defaults to `iqn.2005-10.org.freenas.ctl`.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"isns_servers": schema.SetAttribute{
				Description: "iSNS servers to register the targets with, each an IP address with an " +
					"optional `:port` (port `3205` when none is given). Hostnames are not accepted. " +
					"TrueNAS connects to every server on each update and refuses the update if one " +
					"does not answer. An empty set removes them all. Left out, the current servers " +
					"are kept.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"listen_port": schema.Int64Attribute{
				Description: "TCP port the iSCSI service listens on, 1025 to 65535. Left out, the " +
					"current value is kept; TrueNAS defaults to `3260`.",
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					int64validator.Between(1025, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pool_avail_threshold": schema.Int64Attribute{
				Description: "Pool available space threshold as a percentage, 1 to 99. `0` turns the " +
					"threshold off, which TrueNAS stores as null; a Terraform number cannot be both " +
					"set and null, so `0` is how \"off\" is written. Left out, the current value is " +
					"kept; TrueNAS defaults to off.",
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					int64validator.Between(0, 99),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"alua": schema.BoolAttribute{
				Description: "Whether ALUA (Asymmetric Logical Unit Access) is enabled. Read-only: " +
					"TrueNAS only honors it on Enterprise-licensed high availability systems, and " +
					"on any other system it accepts the flag and ignores it.",
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"iser": schema.BoolAttribute{
				Description: "Whether iSER (iSCSI Extensions for RDMA) is enabled. Read-only: " +
					"TrueNAS only allows it on Enterprise-licensed systems with an RDMA-capable " +
					"network card, and restarts the iSCSI service when it changes.",
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ISCSIGlobalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// updateRequest is the request form of the plan. Settings the plan left
// unset are omitted, because iscsi.global.update is a ForUpdateMetaclass
// model. The exception is isns_servers: that service method rebuilds the
// list from the request alone, so a request without it empties the list.
// The servers are therefore always sent, from the plan when it has them and
// from the current configuration otherwise. They are sorted so the same set
// always sends the same request.
func (r *ISCSIGlobalResource) updateRequest(ctx context.Context, plan *ISCSIGlobalResourceModel, diags *diag.Diagnostics) (*truenas.ISCSIGlobalUpdateRequest, bool) {
	req := &truenas.ISCSIGlobalUpdateRequest{}

	if !plan.Basename.IsNull() && !plan.Basename.IsUnknown() {
		v := plan.Basename.ValueString()
		req.Basename = &v
	}
	if !plan.ListenPort.IsNull() && !plan.ListenPort.IsUnknown() {
		v := int(plan.ListenPort.ValueInt64())
		req.ListenPort = &v
	}
	if !plan.PoolAvailThreshold.IsNull() && !plan.PoolAvailThreshold.IsUnknown() {
		if v := int(plan.PoolAvailThreshold.ValueInt64()); v == 0 {
			req.PoolAvailThreshold = truenas.Null[int]()
		} else {
			req.PoolAvailThreshold = truenas.Value(v)
		}
	}

	if plan.IsnsServers.IsNull() || plan.IsnsServers.IsUnknown() {
		current, err := r.client.GetISCSIGlobal(ctx)
		if err != nil {
			diags.AddError("Error Reading iSCSI Global Config", err.Error())
			return nil, false
		}
		req.IsnsServers = slices.Clone(current.IsnsServers)
	} else {
		diags.Append(plan.IsnsServers.ElementsAs(ctx, &req.IsnsServers, false)...)
	}
	if req.IsnsServers == nil {
		req.IsnsServers = []string{}
	}
	slices.Sort(req.IsnsServers)

	return req, !diags.HasError()
}

// setISCSIGlobalModel populates the model from what TrueNAS returned. A
// null threshold is the empty value 0, so state is never null there.
func setISCSIGlobalModel(model *ISCSIGlobalResourceModel, cfg *truenas.ISCSIGlobal) {
	model.ID = types.StringValue(iscsiGlobalSingletonID)
	model.Basename = types.StringValue(cfg.Basename)
	model.IsnsServers = stringSetValue(cfg.IsnsServers)
	model.ListenPort = types.Int64Value(int64(cfg.ListenPort))
	model.PoolAvailThreshold = types.Int64Value(int64(valueOrZero(cfg.PoolAvailThreshold)))
	model.Alua = types.BoolValue(cfg.Alua)
	model.Iser = types.BoolValue(cfg.Iser)
}

// apply writes the plan and stores what TrueNAS returned. Create and Update
// are the same write, because the configuration always exists.
func (r *ISCSIGlobalResource) apply(ctx context.Context, plan *ISCSIGlobalResourceModel, summary string, diags *diag.Diagnostics) bool {
	req, ok := r.updateRequest(ctx, plan, diags)
	if !ok {
		return false
	}
	cfg, err := r.client.UpdateISCSIGlobal(ctx, req)
	if err != nil {
		diags.AddError(summary, err.Error())
		return false
	}
	setISCSIGlobalModel(plan, cfg)
	return true
}

func (r *ISCSIGlobalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSIGlobal start")

	var plan ISCSIGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	if !r.apply(ctx, &plan, "Error Creating iSCSI Global Config", &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create ISCSIGlobal success")
}

func (r *ISCSIGlobalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSIGlobal start")

	var state ISCSIGlobalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.GetISCSIGlobal(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading iSCSI Global Config", err.Error())
		return
	}

	setISCSIGlobalModel(&state, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read ISCSIGlobal success")
}

func (r *ISCSIGlobalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSIGlobal start")

	var plan ISCSIGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	if !r.apply(ctx, &plan, "Error Updating iSCSI Global Config", &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update ISCSIGlobal success")
}

// Delete is a no-op. The iSCSI global configuration is a singleton that
// always exists; removing the resource stops Terraform managing it. Putting
// the defaults back would rename every target IQN and could move the listen
// port under connected initiators.
func (r *ISCSIGlobalResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSIGlobal: no-op, singleton config is not removable")
}

func (r *ISCSIGlobalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != iscsiGlobalSingletonID {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("truenas_iscsi_global is a singleton; the import ID must be %q, got %q",
				iscsiGlobalSingletonID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
