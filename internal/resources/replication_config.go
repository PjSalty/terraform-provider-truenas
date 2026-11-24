package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// replicationConfigSingletonID is the fixed identifier for
// truenas_replication_config. TrueNAS has exactly one replication
// configuration, so the ID is a constant.
const replicationConfigSingletonID = "replication_config"

var (
	_ resource.Resource                = &ReplicationConfigResource{}
	_ resource.ResourceWithImportState = &ReplicationConfigResource{}
)

// ReplicationConfigResource manages the replication global configuration.
type ReplicationConfigResource struct {
	client *wsclient.Client
}

// ReplicationConfigResourceModel describes the resource data model.
type ReplicationConfigResourceModel struct {
	ID                          types.String   `tfsdk:"id"`
	MaxParallelReplicationTasks types.Int64    `tfsdk:"max_parallel_replication_tasks"`
	Timeouts                    timeouts.Value `tfsdk:"timeouts"`
}

// NewReplicationConfigResource returns a new ReplicationConfigResource
// factory.
func NewReplicationConfigResource() resource.Resource {
	return &ReplicationConfigResource{}
}

func (r *ReplicationConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_replication_config"
}

func (r *ReplicationConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the replication global configuration on TrueNAS SCALE: how many " +
			"replication tasks may run at the same time. This is a singleton, TrueNAS has exactly one.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Fixed singleton identifier. Always \"replication_config\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"max_parallel_replication_tasks": schema.Int64Attribute{
				Description: "Maximum number of replication tasks that run in parallel, at least 1. " +
					"`0` means no limit: TrueNAS stores that as null, and a Terraform number cannot " +
					"be both set and null, so `0` is how \"no limit\" is written. Left out, the " +
					"current value is kept; TrueNAS defaults to `5`.",
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ReplicationConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// updateRequest is the request form of the plan. An unset limit is left
// out, because replication.config.update is a ForUpdateMetaclass model. A
// limit of 0 is sent as null, which removes the limit: it is the empty
// value the resource reads a null back as.
func (r *ReplicationConfigResource) updateRequest(plan *ReplicationConfigResourceModel) *truenas.ReplicationConfigUpdateRequest {
	req := &truenas.ReplicationConfigUpdateRequest{}
	if plan.MaxParallelReplicationTasks.IsNull() || plan.MaxParallelReplicationTasks.IsUnknown() {
		return req
	}
	if v := int(plan.MaxParallelReplicationTasks.ValueInt64()); v == 0 {
		req.MaxParallelReplicationTasks = truenas.Null[int]()
	} else {
		req.MaxParallelReplicationTasks = truenas.Value(v)
	}
	return req
}

// setReplicationConfigModel populates the model from what TrueNAS returned.
// A null limit is the empty value 0, so state is never null there.
func setReplicationConfigModel(model *ReplicationConfigResourceModel, cfg *truenas.ReplicationConfig) {
	model.ID = types.StringValue(replicationConfigSingletonID)
	model.MaxParallelReplicationTasks = types.Int64Value(int64(valueOrZero(cfg.MaxParallelReplicationTasks)))
}

func (r *ReplicationConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ReplicationConfig start")

	var plan ReplicationConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.UpdateReplicationConfig(ctx, r.updateRequest(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Replication Config", err.Error())
		return
	}

	setReplicationConfigModel(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create ReplicationConfig success")
}

func (r *ReplicationConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ReplicationConfig start")

	var state ReplicationConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.GetReplicationConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Replication Config", err.Error())
		return
	}

	setReplicationConfigModel(&state, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read ReplicationConfig success")
}

func (r *ReplicationConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ReplicationConfig start")

	var plan ReplicationConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.UpdateReplicationConfig(ctx, r.updateRequest(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Replication Config", err.Error())
		return
	}

	setReplicationConfigModel(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update ReplicationConfig success")
}

// Delete is a no-op. The replication configuration is a singleton that
// always exists; removing the resource stops Terraform managing it. It does
// not change the limit, because a destroy that raised or removed it would
// start replication tasks that were being held back.
func (r *ReplicationConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ReplicationConfig: no-op, singleton config is not removable")
}

func (r *ReplicationConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != replicationConfigSingletonID {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("truenas_replication_config is a singleton; the import ID must be %q, got %q",
				replicationConfigSingletonID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
