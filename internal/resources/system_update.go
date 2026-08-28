package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

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

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// systemUpdateSingletonID is the fixed identifier used for the
// truenas_system_update resource. TrueNAS has exactly one update
// config per system, so the resource ID is a constant rather than
// a numeric key. ImportState rejects any other value.
const systemUpdateSingletonID = "system_update"

var (
	_ resource.Resource                 = &SystemUpdateResource{}
	_ resource.ResourceWithImportState  = &SystemUpdateResource{}
	_ resource.ResourceWithUpgradeState = &SystemUpdateResource{}
)

// SystemUpdateResource manages the TrueNAS SCALE system update configuration -
// the nightly autocheck toggle and the active update profile. This resource does
// not apply updates; it only governs how the system behaves when an update
// becomes available. Applying an update remains a manual action.
type SystemUpdateResource struct {
	client *wsclient.Client
}

// SystemUpdateResourceModel describes the resource data model.
type SystemUpdateResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	Autocheck        types.Bool     `tfsdk:"autocheck"`
	Profile          types.String   `tfsdk:"profile"`
	CurrentVersion   types.String   `tfsdk:"current_version"`
	Status           types.String   `tfsdk:"status"`
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
		Version: 1,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the TrueNAS SCALE system update configuration: the nightly autocheck " +
			"toggle and the active update profile. This resource is a singleton, TrueNAS has exactly one " +
			"update config per system. It does NOT execute updates; applying an update is a separate " +
			"manual action outside Terraform's control. Use this resource to pin an update profile " +
			"and/or disable the nightly check so that SCALE updates never happen without a " +
			"conscious action.",
		MarkdownDescription: "Manages the TrueNAS SCALE system update configuration: the `autocheck` " +
			"toggle and the active update `profile`. This resource is a singleton, TrueNAS has exactly " +
			"one update config per system. It does **not** execute updates; applying an update is a " +
			"separate manual action outside Terraform's control. Use this resource to pin an update " +
			"profile and/or disable the nightly check so that SCALE updates never happen without a " +
			"conscious action.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Fixed singleton identifier. Always \"system_update\".",
				MarkdownDescription: "Fixed singleton identifier. Always `\"system_update\"`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"autocheck": schema.BoolAttribute{
				Description: "Whether TrueNAS automatically checks for and downloads updates nightly. " +
					"Defaults to false, the conservative value: with it disabled, updates never land " +
					"on the system without an explicit operator action.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"profile": schema.StringAttribute{
				Description: "The update profile this system tracks (for example GENERAL or " +
					"MISSION_CRITICAL). Validated against update.profile_choices at apply time, " +
					"honoring the `available` flag, so an unselectable profile is rejected with the " +
					"valid choices listed rather than failing server-side. When omitted, whatever the " +
					"system already has is preserved and reported.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				Description: "Update subsystem status code reported by TrueNAS: NORMAL or ERROR.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

// applyConfig is the shared write path for Create and Update.
//
// One update.update call carries both fields. The previous implementation
// made two separate writes to methods that do not exist; see the note at the
// top of internal/wsclient/system_update.go.
//
// profile is validated against update.profile_choices first, honoring the
// `available` flag, because middleware refuses a profile that is not
// available and the resulting server-side error names neither the valid
// choices nor which attribute was wrong.
func (r *SystemUpdateResource) applyConfig(ctx context.Context, plan *SystemUpdateResourceModel) error {
	req := &truenas.UpdateConfigUpdateRequest{}

	autocheck := plan.Autocheck.ValueBool()
	req.Autocheck = &autocheck

	if !plan.Profile.IsNull() && !plan.Profile.IsUnknown() && plan.Profile.ValueString() != "" {
		want := plan.Profile.ValueString()
		choices, err := r.client.GetUpdateProfileChoices(ctx)
		if err != nil {
			return fmt.Errorf("fetching update profiles for validation: %w", err)
		}
		choice, ok := choices[want]
		if !ok {
			return fmt.Errorf("update profile %q does not exist; available profiles: %s",
				want, availableProfileNames(choices))
		}
		if !choice.Available {
			return fmt.Errorf("update profile %q exists but is not available on this system; "+
				"available profiles: %s", want, availableProfileNames(choices))
		}
		req.Profile = &want
	}

	if _, err := r.client.SetUpdateConfig(ctx, req); err != nil {
		return fmt.Errorf("applying update config: %w", err)
	}
	return nil
}

// availableProfileNames lists only the selectable profiles, sorted, for a
// diagnostic. Listing unavailable ones would send the operator to a value
// middleware will reject.
func availableProfileNames(choices map[string]truenas.UpdateProfileChoice) string {
	names := make([]string, 0, len(choices))
	for name, c := range choices {
		if c.Available {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none reported as available)"
	}
	return strings.Join(names, ", ")
}

// refreshState populates the model from live TrueNAS state.
func (r *SystemUpdateResource) refreshState(ctx context.Context, model *SystemUpdateResourceModel) error {
	cfg, err := r.client.GetUpdateConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading update config: %w", err)
	}
	model.Autocheck = types.BoolValue(cfg.Autocheck)
	// profile is nullable on the wire. Empty string rather than null keeps
	// the attribute known, so a system with no profile selected does not
	// produce an unknown-value plan on every run.
	if cfg.Profile != nil {
		model.Profile = types.StringValue(*cfg.Profile)
	} else {
		model.Profile = types.StringValue("")
	}

	// update.status replaced update.check_available. Its status and error
	// members are nullable, so nothing here dereferences without checking:
	// a nil status means "no information", not "no update available".
	st, err := r.client.GetUpdateStatus(ctx)
	if err != nil {
		return fmt.Errorf("reading update status: %w", err)
	}
	model.Status = types.StringValue(st.Code)
	model.CurrentVersion = types.StringValue("")
	model.AvailableVersion = types.StringValue("")
	if st.Status != nil {
		model.CurrentVersion = types.StringValue(st.Status.CurrentVersion.Version)
		if st.Status.NewVersion != nil {
			model.AvailableVersion = types.StringValue(st.Status.NewVersion.Version)
		}
	}

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
// TrueNAS has no concept of "deleting" the update config, it always exists,
// it's a system singleton. Destroying the resource therefore leaves the last
// applied auto_download and train settings in place on the system. This
// prevents a surprising reboot-risk vector where `terraform destroy` could
// unintentionally re-enable the nightly check and schedule an upgrade.
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

// systemUpdateSchemaV0 is the historical schema, back when this resource
// modeled auto_download and train against methods that do not exist.
func systemUpdateSchemaV0(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version: 0,
		Blocks:  map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})},
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true},
			"auto_download":     schema.BoolAttribute{Optional: true, Computed: true},
			"train":             schema.StringAttribute{Optional: true, Computed: true},
			"current_version":   schema.StringAttribute{Computed: true},
			"available_status":  schema.StringAttribute{Computed: true},
			"available_version": schema.StringAttribute{Computed: true},
		},
	}
}

// systemUpdateModelV0 is the v0 state shape.
type systemUpdateModelV0 struct {
	ID               types.String   `tfsdk:"id"`
	AutoDownload     types.Bool     `tfsdk:"auto_download"`
	Train            types.String   `tfsdk:"train"`
	CurrentVersion   types.String   `tfsdk:"current_version"`
	AvailableStatus  types.String   `tfsdk:"available_status"`
	AvailableVersion types.String   `tfsdk:"available_version"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// UpgradeState migrates v0 state onto the rewritten schema.
//
// auto_download maps cleanly onto autocheck: same meaning, renamed by
// upstream when the update service became a config service.
//
// train has NO successor. TrueNAS 26.0 replaced release trains with update
// profiles, and a stored train name is not a valid profile. It is therefore
// dropped rather than copied, leaving profile empty so the next refresh
// adopts whatever the system actually has. Copying it across would produce a
// plan that tries to set a profile middleware will reject.
//
// The old state cannot be trusted for the computed fields either, since the
// methods that produced them never existed, so they are left empty for the
// refresh to fill.
func (r *SystemUpdateResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	v0 := systemUpdateSchemaV0(ctx)
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior systemUpdateModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				upgraded := SystemUpdateResourceModel{
					ID:               types.StringValue(systemUpdateSingletonID),
					Autocheck:        prior.AutoDownload,
					Profile:          types.StringValue(""),
					Status:           types.StringValue(""),
					CurrentVersion:   types.StringValue(""),
					AvailableVersion: types.StringValue(""),
					Timeouts:         prior.Timeouts,
				}
				if upgraded.Autocheck.IsNull() || upgraded.Autocheck.IsUnknown() {
					upgraded.Autocheck = types.BoolValue(false)
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
	}
}
