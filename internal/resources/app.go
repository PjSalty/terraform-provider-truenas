package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/customtypes"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/validators"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                     = &AppResource{}
	_ resource.ResourceWithImportState      = &AppResource{}
	_ resource.ResourceWithConfigValidators = &AppResource{}
)

// AppResource manages a TrueNAS SCALE deployed application.
type AppResource struct {
	client *wsclient.Client
}

// AppResourceModel describes the resource data model.
//
// `Values` is a JSON-encoded string so users can pass arbitrary
// chart/values configuration without hard-coding a schema.
type AppResourceModel struct {
	ID               types.String               `tfsdk:"id"`
	AppName          types.String               `tfsdk:"app_name"`
	CatalogApp       types.String               `tfsdk:"catalog_app"`
	CustomCompose    customtypes.NormalizedYAML `tfsdk:"custom_compose"`
	Train            types.String               `tfsdk:"train"`
	Version          types.String               `tfsdk:"version"`
	Values           types.String               `tfsdk:"values"`
	RemoveImages     types.Bool                 `tfsdk:"remove_images"`
	RemoveIxVolumes  types.Bool                 `tfsdk:"remove_ix_volumes"`
	State            types.String               `tfsdk:"state"`
	UpgradeAvailable types.Bool                 `tfsdk:"upgrade_available"`
	HumanVersion     types.String               `tfsdk:"human_version"`
	Timeouts         timeouts.Value             `tfsdk:"timeouts"`
}

func NewAppResource() resource.Resource {
	return &AppResource{}
}

func (r *AppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a deployed application on TrueNAS SCALE (Docker/iX). " +
			"Install is asynchronous, the provider waits for the underlying job to complete. " +
			"Default timeouts: 30m for create/update (app install involves image pulls + chart deployment), 10m for delete.",
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
				Description: "The string ID of the app (equal to app_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_name": schema.StringAttribute{
				Description: "The app name. Must be lowercase alphanumeric with hyphens, " +
					"starting with a letter (e.g. 'my-app'). Immutable after creation.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z][a-z0-9-]*$`),
						"must be lowercase alphanumeric with hyphens, starting with a letter",
					),
					stringvalidator.LengthBetween(1, 63),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catalog_app": schema.StringAttribute{
				Description: "The catalog app slug to install (e.g. 'minio', 'plex'). " +
					"Exactly one of catalog_app or custom_compose must be set. " +
					"Immutable after creation.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"custom_compose": schema.StringAttribute{
				CustomType: customtypes.NormalizedYAMLType{},
				Description: "Raw Docker Compose YAML for a custom app install (custom_app). " +
					"Exactly one of catalog_app or custom_compose must be set. Compose " +
					"content edits apply in place via app.update. Comparison is semantic: " +
					"formatting, comments, key order, and quoting are yours and never " +
					"plan as diffs, while structural drift against the server's stored " +
					"compose (checked on every refresh) surfaces as a normal plan diff. " +
					"Converting an existing app between catalog and custom forces " +
					"replacement.",
				Optional: true,
				Validators: []validator.String{
					validators.YAMLDocument(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						appComposeKindFlip,
						appComposeKindFlipDesc,
						appComposeKindFlipDesc,
					),
				},
			},
			"train": schema.StringAttribute{
				Description: "The catalog train (e.g. 'stable', 'enterprise', 'community', 'test'). " +
					"Immutable after creation.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("stable"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					// UseStateForUnknown must run ahead of RequiresReplace so
					// that omitting the (defaulted) attribute from HCL on a
					// subsequent apply does not plan as Unknown and falsely
					// trigger destroy+create. See certificate.go key_type.
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The app chart version to install (e.g. '1.2.3'). " +
					"Defaults to 'latest'. Immutable after creation, use the TrueNAS " +
					"upgrade workflow for in-place version changes.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("latest"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"values": schema.StringAttribute{
				Description: "JSON-encoded values object passed to the app. Arbitrary chart " +
					"configuration, the provider does not validate structure.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("{}"),
			},
			"remove_images": schema.BoolAttribute{
				Description: "On destroy, remove associated container images (default true).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"remove_ix_volumes": schema.BoolAttribute{
				Description: "On destroy, also remove ix-volumes (default false, DANGEROUS).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Description: "The app runtime state (RUNNING, STOPPED, DEPLOYING, CRASHED).",
				Computed:    true,
			},
			"upgrade_available": schema.BoolAttribute{
				Description: "Whether a newer chart version is available.",
				Computed:    true,
			},
			"human_version": schema.StringAttribute{
				Description: "Human-readable app version string.",
				Computed:    true,
			},
		},
	}
}

// appComposeKindFlipDesc documents the only custom_compose transition
// that forces replacement: converting an existing app between a
// catalog install and a custom compose install. Compose content edits
// are in-place updates.
const appComposeKindFlipDesc = "replaces the app only when converting between catalog and " +
	"custom compose kinds, compose content edits update in place"

// appComposeKindFlip forces replacement only when custom_compose flips
// between null and set, which is a catalog<->custom conversion.
// Content edits (set -> set) update in place.
//
// One import wrinkle: an imported custom app has custom_compose null
// in state because the middleware stores the parsed compose and the
// original string cannot be recovered. So a null -> set transition
// only forces replacement when the state also holds a real catalog_app
// slug, i.e. the app really was a catalog install. Imported custom
// apps take the compose in place on their first plan instead.
func appComposeKindFlip(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.StateValue.IsNull() == req.PlanValue.IsNull() {
		// same kind, content edits are in-place
		return
	}
	if req.StateValue.IsNull() {
		var catalogApp types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("catalog_app"), &catalogApp)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if catalogApp.IsNull() || catalogApp.ValueString() == "" {
			// no catalog slug in state: an imported custom app
			// supplying its compose, not a conversion
			return
		}
	}
	resp.RequiresReplace = true
}

// ConfigValidators enforces the catalog vs custom install shape at
// config-validation time, before any network round-trip: exactly one
// of catalog_app / custom_compose must be set, and the catalog-only
// install knobs (train, version, values) cannot combine with
// custom_compose.
func (r *AppResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("catalog_app"),
			path.MatchRoot("custom_compose"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("custom_compose"),
			path.MatchRoot("train"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("custom_compose"),
			path.MatchRoot("version"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("custom_compose"),
			path.MatchRoot("values"),
		),
	}
}

func (r *AppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create App start")

	var plan AppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var createReq *truenas.AppCreateRequest
	if !plan.CustomCompose.IsNull() {
		// custom compose install: custom_app plus the raw compose
		// string, no catalog fields on the wire
		createReq = &truenas.AppCreateRequest{
			AppName:                   plan.AppName.ValueString(),
			CustomApp:                 true,
			CustomComposeConfigString: plan.CustomCompose.ValueString(),
		}
	} else {
		values, err := decodeValues(plan.Values.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid values JSON", err.Error())
			return
		}
		createReq = &truenas.AppCreateRequest{
			AppName:    plan.AppName.ValueString(),
			CatalogApp: plan.CatalogApp.ValueString(),
			Train:      plan.Train.ValueString(),
			Version:    plan.Version.ValueString(),
			Values:     values,
		}
	}

	tflog.Debug(ctx, "Creating TrueNAS app", map[string]interface{}{
		"app_name":    createReq.AppName,
		"catalog_app": createReq.CatalogApp,
		"train":       createReq.Train,
		"version":     createReq.Version,
		"custom_app":  createReq.CustomApp,
	})

	app, err := r.client.CreateApp(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating App",
			fmt.Sprintf("Could not create app %q: %s", createReq.AppName, err),
		)
		return
	}

	r.mapResponseToModel(app, &plan)
	// plan wins for custom_compose on apply: state keeps the planned
	// string verbatim (same pattern as directory's stat-preserving
	// plan). Drift detection is Read's job, doing it here would risk
	// "inconsistent result after apply" on middleware normalization
	// and would let an app.config hiccup orphan a freshly created app.
	// Defaults for delete-time behavior when user didn't supply them.
	if plan.RemoveImages.IsNull() || plan.RemoveImages.IsUnknown() {
		plan.RemoveImages = types.BoolValue(true)
	}
	if plan.RemoveIxVolumes.IsNull() || plan.RemoveIxVolumes.IsUnknown() {
		plan.RemoveIxVolumes = types.BoolValue(false)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create App success")
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read App start")

	var state AppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApp(ctx, state.ID.ValueString())
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading App",
			fmt.Sprintf("Could not read app %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(app, &state)

	if app.CustomApp {
		// imported custom apps: ImportState seeds catalog_app to ""
		// because the server does not return the original slug. Clear
		// the seed to null so the first plan updates in place instead
		// of tripping catalog_app's RequiresReplace on a "" -> null
		// diff.
		if state.CatalogApp.ValueString() == "" {
			state.CatalogApp = types.StringNull()
		}
		// semantic drift check against the server's stored compose;
		// also populates custom_compose after an import
		cfg, err := r.client.GetAppConfig(ctx, state.ID.ValueString())
		if err != nil {
			if wsclient.IsNotFound(err) {
				// app vanished between get_instance and app.config,
				// normal Read remove-from-state flow
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError(
				"Error Reading App",
				fmt.Sprintf("Could not read app %q compose config: %s", state.ID.ValueString(), err),
			)
			return
		}
		r.applyComposeDrift(&state, cfg, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read App success")
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update App start")

	var plan AppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AppResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &truenas.AppUpdateRequest{}
	if !plan.CustomCompose.IsNull() {
		// compose content edits are in-place, the middleware re-parses
		// the string on app.update
		updateReq.CustomComposeConfigString = plan.CustomCompose.ValueString()
	} else {
		values, err := decodeValues(plan.Values.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid values JSON", err.Error())
			return
		}
		updateReq.Values = values
	}

	app, err := r.client.UpdateApp(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating App",
			fmt.Sprintf("Could not update app %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(app, &plan)
	// plan wins for custom_compose on apply, drift detection is
	// Read's job (see Create)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update App success")
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete App start")

	var state AppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	delReq := &truenas.AppDeleteRequest{
		RemoveImages:    boolOrDefault(state.RemoveImages, true),
		RemoveIxVolumes: boolOrDefault(state.RemoveIxVolumes, false),
		// custom apps with a broken or missing compose fail a plain
		// app.delete, force so destroy always succeeds
		ForceRemoveCustomApp: !state.CustomCompose.IsNull(),
	}

	tflog.Debug(ctx, "Deleting TrueNAS app", map[string]interface{}{
		"id":                state.ID.ValueString(),
		"remove_images":     delReq.RemoveImages,
		"remove_ix_volumes": delReq.RemoveIxVolumes,
	})

	if err := r.client.DeleteApp(ctx, state.ID.ValueString(), delReq); err != nil {
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "App already deleted, removing from state", map[string]interface{}{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting App",
			fmt.Sprintf("Could not delete app %q: %s", state.ID.ValueString(), err),
		)
		return
	}
	tflog.Trace(ctx, "Delete App success")
}

// ImportState seeds the non-recoverable attributes with neutral
// defaults. catalog_app is seeded "" because the server does not
// return the original slug; for custom apps the first Read clears it
// to null and fills custom_compose with a canonical dump of the
// server's stored compose (see reconcileCustomCompose), so a config
// that matches the server structurally plans clean right after
// import.
func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("values"), types.StringValue("{}"))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remove_images"), types.BoolValue(true))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remove_ix_volumes"), types.BoolValue(false))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_app"), types.StringValue(""))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("train"), types.StringValue("stable"))...)
}

// applyComposeDrift implements semantic drift detection for custom
// apps on Read: compare the state-held compose string against the
// server's parsed document (app.config) and keep whichever
// representation is truthful, the user's string when structurally
// equal (formatting, comments, key order, quoting, and YAML 1.1 bool
// spellings are the user's business), a canonical dump of the server
// document when not, so the next plan shows the drift. Also populates
// custom_compose after an import, where the state starts null.
func (r *AppResource) applyComposeDrift(model *AppResourceModel, cfg map[string]interface{}, diags *diag.Diagnostics) {
	if len(cfg) == 0 {
		// an empty server config cannot be expressed as a valid
		// custom_compose (the validator requires a non-empty
		// mapping); keep the state value rather than writing one the
		// user could never configure
		return
	}
	dump, err := renderComposeCanonical(cfg)
	if err != nil {
		diags.AddError(
			"Error Reading App",
			fmt.Sprintf("Could not render app %q compose config as YAML: %s", model.ID.ValueString(), err),
		)
		return
	}
	if !model.CustomCompose.IsNull() {
		eq, eqErr := customtypes.YAMLStringsSemanticallyEqual(model.CustomCompose.ValueString(), dump)
		if eqErr == nil && eq {
			return
		}
	}
	model.CustomCompose = customtypes.NewNormalizedYAMLValue(dump)
}

// renderComposeCanonical is a seam over canonicalComposeYAML so the
// marshal-error branch, unreachable through JSON-shaped configs from the
// real client, stays testable.
var renderComposeCanonical = canonicalComposeYAML

// canonicalComposeYAML renders the server's parsed compose document
// as deterministic YAML (yaml.v3 sorts map keys). The recover guard
// converts yaml.Marshal's panic-on-unsupported-type into an error.
func canonicalComposeYAML(doc map[string]interface{}) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rendering compose YAML: %v", r)
		}
	}()
	b, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// mapResponseToModel copies the refetched app onto the model. It
// deliberately never touches CustomCompose: drift handling for the
// compose string is semantic and lives in reconcileCustomCompose.
func (r *AppResource) mapResponseToModel(app *truenas.App, model *AppResourceModel) {
	model.ID = types.StringValue(app.ID)
	model.AppName = types.StringValue(app.Name)
	model.State = types.StringValue(app.State)
	model.UpgradeAvailable = types.BoolValue(app.UpgradeAvailable)
	model.HumanVersion = types.StringValue(app.HumanVersion)
	// version handling: if the user asked for "latest" (or left it
	// unknown/null), we preserve that in state, the resolved concrete
	// version is exposed via human_version. Only when the user pinned a
	// specific version do we store the server's reported value (which
	// should match what they asked for).
	userAskedLatest := model.Version.IsNull() || model.Version.IsUnknown() ||
		model.Version.ValueString() == "" || model.Version.ValueString() == "latest"
	if userAskedLatest {
		model.Version = types.StringValue("latest")
	} else if app.Version != "" {
		model.Version = types.StringValue(app.Version)
	}
}

// decodeValues parses the JSON-encoded values string into a map.
// An empty string is treated as "{}".
func decodeValues(s string) (map[string]interface{}, error) {
	if s == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("values must be a JSON object: %w", err)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}

func boolOrDefault(v types.Bool, def bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueBool()
}
