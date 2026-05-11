package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// systemUpdateSingletonID is the fixed identifier used for the
// truenas_system_update resource. TrueNAS has exactly one update
// config per system, so the resource ID is a constant rather than
// a numeric key. ImportState rejects any other value.
const systemUpdateSingletonID = "system_update"

var (
	_ resource.Resource                = &SystemUpdateResource{}
	_ resource.ResourceWithImportState = &SystemUpdateResource{}
)

// SystemUpdateResource manages the TrueNAS SCALE system update configuration —
// the auto-download toggle and the active release train. This resource does
// not apply updates; it only governs how the system behaves when an update
// becomes available. Applying an update remains a manual action.
type SystemUpdateResource struct {
	client *client.Client
}

// SystemUpdateResourceModel describes the resource data model.
type SystemUpdateResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	AutoDownload     types.Bool     `tfsdk:"auto_download"`
	Train            types.String   `tfsdk:"train"`
	CurrentVersion   types.String   `tfsdk:"current_version"`
	AvailableStatus  types.String   `tfsdk:"available_status"`
	AvailableVersion types.String   `tfsdk:"available_version"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// NewSystemUpdateResource returns a new SystemUpdateResource factory.
func NewSystemUpdateResource() resource.Resource {
	return &SystemUpdateResource{}
}

func (r *SystemUpdateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_update"
}

func (r *SystemUpdateResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the TrueNAS SCALE system update configuration: the auto-download toggle " +
			"and the active release train. This resource is a singleton — TrueNAS has exactly one " +
			"update config per system. It does NOT execute updates; applying an update is a separate " +
			"manual action outside Terraform's control. Use this resource to pin a train and/or " +
			"disable auto-download so that SCALE updates never happen without a conscious action.",
		MarkdownDescription: "Manages the TrueNAS SCALE system update configuration: the `auto_download` " +
			"toggle and the active release `train`. This resource is a singleton — TrueNAS has exactly " +
			"one update config per system. It does **not** execute updates; applying an update is a " +
			"separate manual action outside Terraform's control. Use this resource to pin a train " +
			"and/or disable auto-download so that SCALE updates never happen without a conscious action.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Fixed singleton identifier. Always \"system_update\".",
				MarkdownDescription: "Fixed singleton identifier. Always `\"system_update\"`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_download": schema.BoolAttribute{
				Description: "Whether TrueNAS should automatically download available updates into " +
					"the local update cache. Defaults to false — the conservative pinning value. " +
					"With auto_download disabled, updates never land on the system without an " +
					"explicit operator action.",
				MarkdownDescription: "Whether TrueNAS should automatically download available updates " +
					"into the local update cache. Defaults to `false` — the conservative pinning value. " +
					"With `auto_download` disabled, updates never land on the system without an " +
					"explicit operator action.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"train": schema.StringAttribute{
				Description: "The active release train (e.g., TrueNAS-SCALE-Fangtooth). When set, " +
					"Terraform reconciles the selected train on every apply. When omitted, " +
					"Terraform preserves whatever the system has configured and reports it as a " +
					"computed attribute. Validated against the list returned by the TrueNAS API at " +
					"apply time.",
				MarkdownDescription: "The active release train (e.g., `TrueNAS-SCALE-Fangtooth`). " +
					"When set, Terraform reconciles the selected train on every apply. When omitted, " +
					"Terraform preserves whatever the system has configured and reports it as a " +
					"computed attribute. Validated against the list returned by the TrueNAS API at " +
					"apply time.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"current_version": schema.StringAttribute{
				Description: "The version of TrueNAS SCALE currently running on the system. " +
					"Refreshed from /system/info on every Read. Changes when a SCALE update has " +
					"been applied and the system has rebooted.",
				MarkdownDescription: "The version of TrueNAS SCALE currently running on the system. " +
					"Refreshed from `/system/info` on every Read. Changes when a SCALE update has " +
					"been applied and the system has rebooted.",
				Computed: true,
			},
			"available_status": schema.StringAttribute{
				Description: "The pending-update status reported by the TrueNAS update server. " +
					"One of AVAILABLE, UNAVAILABLE, REBOOT_REQUIRED, HA_UNAVAILABLE. " +
					"UNAVAILABLE is the normal steady-state value.",
				MarkdownDescription: "The pending-update status reported by the TrueNAS update server. " +
					"One of `AVAILABLE`, `UNAVAILABLE`, `REBOOT_REQUIRED`, `HA_UNAVAILABLE`. " +
					"`UNAVAILABLE` is the normal steady-state value.",
				Computed: true,
			},
			"available_version": schema.StringAttribute{
				Description: "When available_status is AVAILABLE, this is the version string of " +
					"the pending update. Empty in all other states.",
				MarkdownDescription: "When `available_status` is `AVAILABLE`, this is the version string " +
					"of the pending update. Empty in all other states.",
				Computed: true,
			},
		},
	}
}

func (r *SystemUpdateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// applyConfig is the shared write path for both Create and Update. It posts
// the user-supplied auto_download and (optionally) train, validating the
// train against the live get_trains list first. Both writes are best-effort
// ordered: train first, then auto_download, so a failure on auto_download
// does not leave a mismatched train in state.
func (r *SystemUpdateResource) applyConfig(ctx context.Context, plan *SystemUpdateResourceModel) error {
	if !plan.Train.IsNull() && !plan.Train.IsUnknown() {
		trains, err := r.client.GetUpdateTrains(ctx)
		if err != nil {
			return fmt.Errorf("fetching update trains for validation: %w", err)
		}
		want := plan.Train.ValueString()
		if _, ok := trains.Trains[want]; !ok {
			available := make([]string, 0, len(trains.Trains))
			for name := range trains.Trains {
				available = append(available, name)
			}
			return fmt.Errorf("train %q not found in available trains: %v", want, available)
		}
		if trains.Selected != want {
			if err := r.client.SetUpdateTrain(ctx, want); err != nil {
				return fmt.Errorf("setting train: %w", err)
			}
		}
	}

	if err := r.client.SetUpdateAutoDownload(ctx, plan.AutoDownload.ValueBool()); err != nil {
		return fmt.Errorf("setting auto_download: %w", err)
	}

	return nil
}

// refreshState populates every field on the model from live TrueNAS state.
// Called by Read, Create, and Update so all three have a single source of
// truth for what the post-write model should look like.
func (r *SystemUpdateResource) refreshState(ctx context.Context, model *SystemUpdateResourceModel) error {
	autoDownload, err := r.client.GetUpdateAutoDownload(ctx)
	if err != nil {
		return fmt.Errorf("reading auto_download: %w", err)
	}
	model.AutoDownload = types.BoolValue(autoDownload)

	trains, err := r.client.GetUpdateTrains(ctx)
	if err != nil {
		return fmt.Errorf("reading update trains: %w", err)
	}
	model.Train = types.StringValue(trains.Selected)

	info, err := r.client.GetSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("reading system info: %w", err)
	}
	model.CurrentVersion = types.StringValue(info.Version)

	check, err := r.client.CheckUpdateAvailable(ctx)
	if err != nil {
		return fmt.Errorf("checking update availability: %w", err)
	}
	model.AvailableStatus = types.StringValue(check.Status)
	model.AvailableVersion = types.StringValue(check.Version)

	model.ID = types.StringValue(systemUpdateSingletonID)
	return nil
}

func (r *SystemUpdateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SystemUpdate start")

	var plan SystemUpdateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Applying System Update Config", err.Error())
		return
	}

	if err := r.refreshState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Refreshing System Update State", err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SystemUpdate success")
}

func (r *SystemUpdateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SystemUpdate start")

	var state SystemUpdateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refreshState(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error Reading System Update", err.Error())
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SystemUpdate success")
}

func (r *SystemUpdateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SystemUpdate start")

	var plan SystemUpdateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Updating System Update Config", err.Error())
		return
	}

	if err := r.refreshState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Refreshing System Update State", err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SystemUpdate success")
}

// Delete is a no-op that only removes the resource from Terraform state.
// TrueNAS has no concept of "deleting" the update config — it always exists,
// it's a system singleton. Destroying the resource therefore leaves the last
// applied auto_download and train settings in place on the system. This
// prevents a surprising reboot-risk vector where `terraform destroy` could
// unintentionally re-enable auto-download and schedule an upgrade.
func (r *SystemUpdateResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SystemUpdate no-op (singleton)")
}

// ImportState accepts exactly the constant "system_update" as the import ID
// and rejects anything else. This is a singleton: there is nothing to
// disambiguate, and accepting arbitrary IDs would invite operator confusion.
func (r *SystemUpdateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != systemUpdateSingletonID {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("truenas_system_update is a singleton; the only valid import ID is %q. Got: %q.",
				systemUpdateSingletonID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
