package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &AppResource{}
	_ resource.ResourceWithImportState = &AppResource{}
)

// AppResource manages a TrueNAS SCALE deployed application.
type AppResource struct {
	client *client.Client
}

// AppResourceModel describes the resource data model.
//
// `Values` is a JSON-encoded string so users can pass arbitrary
// chart/values configuration without hard-coding a schema.
type AppResourceModel struct {
	ID               types.String   `tfsdk:"id"`
	AppName          types.String   `tfsdk:"app_name"`
	CatalogApp       types.String   `tfsdk:"catalog_app"`
	Train            types.String   `tfsdk:"train"`
	Version          types.String   `tfsdk:"version"`
	Values           types.String   `tfsdk:"values"`
	RemoveImages     types.Bool     `tfsdk:"remove_images"`
	RemoveIxVolumes  types.Bool     `tfsdk:"remove_ix_volumes"`
	State            types.String   `tfsdk:"state"`
	UpgradeAvailable types.Bool     `tfsdk:"upgrade_available"`
	HumanVersion     types.String   `tfsdk:"human_version"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
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
			"Install is asynchronous — the provider waits for the underlying job to complete. " +
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
					"Immutable after creation.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
					"Defaults to 'latest'. Immutable after creation — use the TrueNAS " +
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
					"configuration — the provider does not validate structure.",
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
				Description: "On destroy, also remove ix-volumes (default false — DANGEROUS).",
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

func (r *AppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create App start")

	var plan AppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	values, err := decodeValues(plan.Values.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid values JSON", err.Error())
		return
	}

	createReq := &client.AppCreateRequest{
		AppName:    plan.AppName.ValueString(),
		CatalogApp: plan.CatalogApp.ValueString(),
		Train:      plan.Train.ValueString(),
		Version:    plan.Version.ValueString(),
		Values:     values,
	}

	tflog.Debug(ctx, "Creating TrueNAS app", map[string]interface{}{
		"app_name":    createReq.AppName,
		"catalog_app": createReq.CatalogApp,
		"train":       createReq.Train,
		"version":     createReq.Version,
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
		if client.IsNotFound(err) {
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

	values, err := decodeValues(plan.Values.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid values JSON", err.Error())
		return
	}

	updateReq := &client.AppUpdateRequest{
		Values: values,
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

	delReq := &client.AppDeleteRequest{
		RemoveImages:    boolOrDefault(state.RemoveImages, true),
		RemoveIxVolumes: boolOrDefault(state.RemoveIxVolumes, false),
	}

	tflog.Debug(ctx, "Deleting TrueNAS app", map[string]interface{}{
		"id":                state.ID.ValueString(),
		"remove_images":     delReq.RemoveImages,
		"remove_ix_volumes": delReq.RemoveIxVolumes,
	})

	if err := r.client.DeleteApp(ctx, state.ID.ValueString(), delReq); err != nil {
		if client.IsNotFound(err) {
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

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("values"), types.StringValue("{}"))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remove_images"), types.BoolValue(true))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remove_ix_volumes"), types.BoolValue(false))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("catalog_app"), types.StringValue(""))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("train"), types.StringValue("stable"))...)
}

func (r *AppResource) mapResponseToModel(app *client.App, model *AppResourceModel) {
	model.ID = types.StringValue(app.ID)
	model.AppName = types.StringValue(app.Name)
	model.State = types.StringValue(app.State)
	model.UpgradeAvailable = types.BoolValue(app.UpgradeAvailable)
	model.HumanVersion = types.StringValue(app.HumanVersion)
	// version handling: if the user asked for "latest" (or left it
	// unknown/null), we preserve that in state — the resolved concrete
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
