package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &PoolResource{}
	_ resource.ResourceWithImportState = &PoolResource{}
	_ resource.ResourceWithModifyPlan  = &PoolResource{}
)

// PoolResource manages the lifecycle of a ZFS pool on TrueNAS SCALE.
//
// The TrueNAS /pool topology schema is a deeply-nested structure with
// discriminated unions (MIRROR/RAIDZ*/DRAID*/STRIPE) per vdev class
// (data/cache/log/spares/special/dedup). Modeling that as a nested
// Terraform Block would require dozens of schema attributes and a lot
// of translation code for very little gain since pool creation is
// typically a one-shot operation. We instead accept `topology_json` as
// a raw JSON string that is passed through to the API verbatim, with
// the tradeoff that schema validation happens server-side.
type PoolResource struct {
	client *client.Client
}

// PoolResourceModel describes the pool resource data model.
type PoolResourceModel struct {
	ID                    types.String   `tfsdk:"id"`
	Name                  types.String   `tfsdk:"name"`
	TopologyJSON          types.String   `tfsdk:"topology_json"`
	Encryption            types.Bool     `tfsdk:"encryption"`
	EncryptionOptionsJSON types.String   `tfsdk:"encryption_options_json"`
	Deduplication         types.String   `tfsdk:"deduplication"`
	Checksum              types.String   `tfsdk:"checksum"`
	AllowDuplicateSerials types.Bool     `tfsdk:"allow_duplicate_serials"`
	GUID                  types.String   `tfsdk:"guid"`
	Path                  types.String   `tfsdk:"path"`
	Status                types.String   `tfsdk:"status"`
	Healthy               types.Bool     `tfsdk:"healthy"`
	Timeouts              timeouts.Value `tfsdk:"timeouts"`
}

func NewPoolResource() resource.Resource {
	return &PoolResource{}
}

func (r *PoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (r *PoolResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ZFS pool on TrueNAS SCALE. Creation is asynchronous and " +
			"the topology is passed through to the API as a raw JSON object to avoid " +
			"modeling the deeply-nested discriminated-union schema. " +
			"Default timeouts: 60m for create (zpool creation + encryption + initial scrub can be slow), 30m for update/delete." + "\n\n" +
			"**Stability: Beta.** Import and read verified live. Full create/destroy cycle has not been observed because the test environment has a single pool that cannot be destroyed. Pool creation uses a deeply-nested topology schema expressed as JSON; the format matches the TrueNAS REST API.",
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
				Description: "The numeric ID of the pool, assigned by TrueNAS.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the pool (1-50 characters).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"topology_json": schema.StringAttribute{
				Description: "The pool topology as a raw JSON object. Must contain at least a " +
					"`data` key with a list of vdev definitions (e.g. " +
					`{"data":[{"type":"MIRROR","disks":["sda","sdb"]}]}` + "). " +
					"May also contain cache, log, spares, special, and dedup keys. " +
					"Required on create; ignored after import since the API does " +
					"not round-trip the original request form.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"encryption": schema.BoolAttribute{
				Description: "Whether to create a ZFS-encrypted root dataset for this pool.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"encryption_options_json": schema.StringAttribute{
				Description: "Optional encryption options as a raw JSON object " +
					"(e.g. generate_key, algorithm, passphrase, key, pbkdf2iters).",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deduplication": schema.StringAttribute{
				Description: "Deduplication mode: ON, VERIFY, OFF, or unset.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ON", "VERIFY", "OFF"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checksum": schema.StringAttribute{
				Description: "Checksum algorithm: ON, OFF, FLETCHER2, FLETCHER4, SHA256, SHA512, SKEIN, EDONR, BLAKE3.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ON", "OFF", "FLETCHER2", "FLETCHER4", "SHA256", "SHA512", "SKEIN", "EDONR", "BLAKE3"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_duplicate_serials": schema.BoolAttribute{
				Description: "Whether to allow disks with duplicate serial numbers in this pool.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"guid": schema.StringAttribute{
				Description: "The ZFS GUID of the pool.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The filesystem mount path of the pool.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the pool (ONLINE, DEGRADED, FAULTED, ...).",
				Computed:    true,
			},
			"healthy": schema.BoolAttribute{
				Description: "Whether the pool is in a healthy state.",
				Computed:    true,
			},
		},
	}
}

func (r *PoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Pool start")

	var plan PoolResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate the raw JSON up-front so Terraform fails with a clean error
	// before hitting the API.
	var topology json.RawMessage
	if err := json.Unmarshal([]byte(plan.TopologyJSON.ValueString()), &topology); err != nil {
		resp.Diagnostics.AddError(
			"Invalid topology_json",
			fmt.Sprintf("topology_json must be valid JSON: %s", err),
		)
		return
	}

	createReq := &client.PoolCreateRequest{
		Name:     plan.Name.ValueString(),
		Topology: topology,
	}
	if !plan.Encryption.IsNull() && !plan.Encryption.IsUnknown() {
		createReq.Encryption = plan.Encryption.ValueBool()
	}
	if !plan.EncryptionOptionsJSON.IsNull() && !plan.EncryptionOptionsJSON.IsUnknown() && plan.EncryptionOptionsJSON.ValueString() != "" {
		var opts map[string]interface{}
		if err := json.Unmarshal([]byte(plan.EncryptionOptionsJSON.ValueString()), &opts); err != nil {
			resp.Diagnostics.AddError(
				"Invalid encryption_options_json",
				fmt.Sprintf("encryption_options_json must be valid JSON object: %s", err),
			)
			return
		}
		createReq.EncryptionOptions = opts
	}
	if !plan.Deduplication.IsNull() && !plan.Deduplication.IsUnknown() {
		createReq.Deduplication = plan.Deduplication.ValueString()
	}
	if !plan.Checksum.IsNull() && !plan.Checksum.IsUnknown() {
		createReq.Checksum = plan.Checksum.ValueString()
	}
	if !plan.AllowDuplicateSerials.IsNull() && !plan.AllowDuplicateSerials.IsUnknown() {
		createReq.AllowDuplicateSerials = plan.AllowDuplicateSerials.ValueBool()
	}

	tflog.Debug(ctx, "Creating pool", map[string]interface{}{"name": createReq.Name})

	pool, err := r.client.CreatePool(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Pool",
			fmt.Sprintf("Could not create pool %q: %s", createReq.Name, err),
		)
		return
	}

	r.mapResponseToModel(pool, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Pool success")
}

func (r *PoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Pool start")

	var state PoolResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse pool ID: %s", err))
		return
	}

	pool, err := r.client.GetPool(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Pool",
			fmt.Sprintf("Could not read pool %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(pool, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Pool success")
}

func (r *PoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Pool start")

	// The pool resource has RequiresReplace on all mutable fields, so Update
	// is effectively a no-op — Terraform will destroy+recreate on change.
	// We still need to carry forward the planned state.
	var plan PoolResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Pool success")
}

func (r *PoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Pool start")

	var state PoolResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse pool ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Exporting/destroying pool", map[string]interface{}{"id": id})

	err = r.client.ExportPool(ctx, id, &client.PoolExportRequest{
		Cascade:         true,
		RestartServices: false,
		Destroy:         true,
	})
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Pool already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Destroying Pool",
			fmt.Sprintf("Could not destroy pool %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Pool success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *PoolResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_pool")
}

func (r *PoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Validate ID is numeric before passing through.
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Pool ID must be numeric: %s", err))
		return
	}
	// Delegate to the standard passthrough helper so the framework sets up
	// the `id` attribute and a properly-typed null `timeouts` block. Read
	// is called afterward by the framework to populate the rest of state.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PoolResource) mapResponseToModel(pool *client.Pool, model *PoolResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(pool.ID))
	model.Name = types.StringValue(pool.Name)
	model.GUID = types.StringValue(pool.GUID)
	model.Path = types.StringValue(pool.Path)
	model.Status = types.StringValue(pool.Status)
	model.Healthy = types.BoolValue(pool.Healthy)

	// Optional+Computed fields: populate with defaults if unset
	if model.Encryption.IsNull() || model.Encryption.IsUnknown() {
		model.Encryption = types.BoolValue(false)
	}
	if model.Deduplication.IsNull() || model.Deduplication.IsUnknown() {
		model.Deduplication = types.StringValue("")
	}
	if model.Checksum.IsNull() || model.Checksum.IsUnknown() {
		model.Checksum = types.StringValue("")
	}
	if model.AllowDuplicateSerials.IsNull() || model.AllowDuplicateSerials.IsUnknown() {
		model.AllowDuplicateSerials = types.BoolValue(false)
	}
}
