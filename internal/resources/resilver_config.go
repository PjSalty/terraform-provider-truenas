package resources

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

// resilverConfigSingletonID is the fixed identifier for
// truenas_resilver_config. TrueNAS has exactly one resilver priority
// window, so the ID is a constant.
const resilverConfigSingletonID = "resilver_config"

// resilverTimePattern is a zero-padded 24 hour "HH:MM". TrueNAS also
// accepts "9:00" on a write, but it reads every time back as "%H:%M", so
// the unpadded form would show a diff on every plan.
var resilverTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

const resilverTimeMessage = `must be a zero-padded 24 hour time such as "18:00" or "09:00"`

var (
	_ resource.Resource                = &ResilverConfigResource{}
	_ resource.ResourceWithImportState = &ResilverConfigResource{}
)

// ResilverConfigResource manages the pool resilver priority window.
type ResilverConfigResource struct {
	client *wsclient.Client
}

// ResilverConfigResourceModel describes the resource data model.
type ResilverConfigResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Enabled  types.Bool     `tfsdk:"enabled"`
	Begin    types.String   `tfsdk:"begin"`
	End      types.String   `tfsdk:"end"`
	Weekday  types.Set      `tfsdk:"weekday"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// NewResilverConfigResource returns a new ResilverConfigResource factory.
func NewResilverConfigResource() resource.Resource {
	return &ResilverConfigResource{}
}

func (r *ResilverConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resilver_config"
}

func (r *ResilverConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the pool resilver priority window on TrueNAS SCALE. Inside the window TrueNAS " +
			"raises the ZFS resilver tunables, so a resilver gets more of the disks and finishes sooner; " +
			"outside it the defaults apply. This is a singleton, TrueNAS has exactly one window, and it " +
			"covers every pool.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Fixed singleton identifier. Always \"resilver_config\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the priority window is in force. When false TrueNAS stops changing " +
					"the resilver tunables and leaves them as they are, whatever the window says. Left " +
					"out, the current value is kept.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"begin": schema.StringAttribute{
				Description: "Start of the window as a zero-padded 24 hour time, `HH:MM`, in the " +
					"system time zone. A `begin` later than `end` rolls over midnight. Left out, the " +
					"current value is kept; TrueNAS defaults to `18:00`.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(resilverTimePattern, resilverTimeMessage),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"end": schema.StringAttribute{
				Description: "End of the window as a zero-padded 24 hour time, `HH:MM`, in the " +
					"system time zone. Left out, the current value is kept; TrueNAS defaults to `09:00`.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(resilverTimePattern, resilverTimeMessage),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"weekday": schema.SetAttribute{
				Description: "Days the window begins on, `1` (Monday) through `7` (Sunday). A window that " +
					"rolls over midnight belongs to the day it begins on and runs into the next morning, " +
					"whether or not that next day is listed. At least one day is required. Left out, the " +
					"current value is kept; TrueNAS defaults to every day.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueInt64sAre(int64validator.Between(1, 7)),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ResilverConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// updateRequest is the request form of the plan. Only attributes the plan
// set are included: pool.resilver.update is a ForUpdateMetaclass model, so
// an omitted key leaves the stored value alone. The days are sorted, so the
// same set always sends the same request.
func (r *ResilverConfigResource) updateRequest(ctx context.Context, plan *ResilverConfigResourceModel, diags *diag.Diagnostics) *truenas.ResilverConfigUpdateRequest {
	req := &truenas.ResilverConfigUpdateRequest{}

	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		req.Enabled = &v
	}
	if !plan.Begin.IsNull() && !plan.Begin.IsUnknown() {
		v := plan.Begin.ValueString()
		req.Begin = &v
	}
	if !plan.End.IsNull() && !plan.End.IsUnknown() {
		v := plan.End.ValueString()
		req.End = &v
	}
	if !plan.Weekday.IsNull() && !plan.Weekday.IsUnknown() {
		var planned []int64
		diags.Append(plan.Weekday.ElementsAs(ctx, &planned, false)...)
		req.Weekday = make([]int, 0, len(planned))
		for _, day := range planned {
			req.Weekday = append(req.Weekday, int(day))
		}
		slices.Sort(req.Weekday)
	}

	return req
}

// setModel populates the model from what TrueNAS returned.
func setModel(model *ResilverConfigResourceModel, cfg *truenas.ResilverConfig) {
	days := make([]attr.Value, 0, len(cfg.Weekday))
	for _, day := range cfg.Weekday {
		days = append(days, types.Int64Value(int64(day)))
	}

	model.ID = types.StringValue(resilverConfigSingletonID)
	model.Enabled = types.BoolValue(cfg.Enabled)
	model.Begin = types.StringValue(cfg.Begin)
	model.End = types.StringValue(cfg.End)
	model.Weekday = types.SetValueMust(types.Int64Type, days)
}

func (r *ResilverConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ResilverConfig start")

	var plan ResilverConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.UpdateResilverConfig(ctx, r.updateRequest(ctx, &plan, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resilver Config", err.Error())
		return
	}

	setModel(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create ResilverConfig success")
}

func (r *ResilverConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ResilverConfig start")

	var state ResilverConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.GetResilverConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resilver Config", err.Error())
		return
	}

	setModel(&state, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read ResilverConfig success")
}

func (r *ResilverConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ResilverConfig start")

	var plan ResilverConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	cfg, err := r.client.UpdateResilverConfig(ctx, r.updateRequest(ctx, &plan, &resp.Diagnostics))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resilver Config", err.Error())
		return
	}

	setModel(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update ResilverConfig success")
}

// Delete is a no-op. The resilver priority window is a singleton that
// always exists; removing the resource stops Terraform managing it. It does
// not switch the window off, because a destroy that slowed a running
// resilver down would be a surprise on a degraded pool.
func (r *ResilverConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ResilverConfig: no-op, singleton config is not removable")
}

func (r *ResilverConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != resilverConfigSingletonID {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("truenas_resilver_config is a singleton; the import ID must be %q, got %q",
				resilverConfigSingletonID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
