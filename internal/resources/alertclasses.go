// alertclasses.go
//
// Singleton resource. Using a nested Map attribute rather than a JSON string
// because every map value has the same well-typed shape
// ({level, policy, proactive_support}) across all alert classes — this gives
// users proper validation and IDE completion without jsonencode boilerplate.
package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &AlertClassesResource{}
	_ resource.ResourceWithImportState = &AlertClassesResource{}
)

type AlertClassesResource struct {
	client *client.Client
}

type AlertClassesResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Classes  types.Map      `tfsdk:"classes"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type AlertClassEntryModel struct {
	Level            types.String `tfsdk:"level"`
	Policy           types.String `tfsdk:"policy"`
	ProactiveSupport types.Bool   `tfsdk:"proactive_support"`
}

var alertClassEntryAttrTypes = map[string]attr.Type{
	"level":             types.StringType,
	"policy":            types.StringType,
	"proactive_support": types.BoolType,
}

func NewAlertClassesResource() resource.Resource {
	return &AlertClassesResource{}
}

func (r *AlertClassesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alertclasses"
}

func (r *AlertClassesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the singleton alert class severity/notification-policy configuration on TrueNAS SCALE.",
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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"classes": schema.MapNestedAttribute{
				Description: "Map of alert class name to configuration.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"level": schema.StringAttribute{
							Description: "Severity level: INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("INFO", "NOTICE", "WARNING", "ERROR", "CRITICAL", "ALERT", "EMERGENCY"),
							},
						},
						"policy": schema.StringAttribute{
							Description: "Notification policy: IMMEDIATELY, HOURLY, DAILY, NEVER.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("IMMEDIATELY", "HOURLY", "DAILY", "NEVER"),
							},
						},
						"proactive_support": schema.BoolAttribute{
							Description: "Include alerts of this class in proactive support reporting.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *AlertClassesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AlertClassesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create AlertClasses start")

	var plan AlertClassesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating alertclasses config (updating singleton)")

	updateReq := r.buildUpdateRequest(ctx, &plan)

	cfg, err := r.client.UpdateAlertClassesConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Alert Classes", fmt.Sprintf("Could not update alert classes: %s", err))
		return
	}

	r.mapResponseToModel(ctx, cfg, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create AlertClasses success")
}

func (r *AlertClassesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read AlertClasses start")

	var state AlertClassesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetAlertClassesConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Alert Classes", fmt.Sprintf("Could not read alert classes: %s", err))
		return
	}

	r.mapResponseToModel(ctx, cfg, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read AlertClasses success")
}

func (r *AlertClassesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update AlertClasses start")

	var plan AlertClassesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(ctx, &plan)

	cfg, err := r.client.UpdateAlertClassesConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Alert Classes", fmt.Sprintf("Could not update alert classes: %s", err))
		return
	}

	r.mapResponseToModel(ctx, cfg, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update AlertClasses success")
}

func (r *AlertClassesResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete AlertClasses start")

	tflog.Debug(ctx, "Deleting alertclasses config (resetting to empty)")

	_, err := r.client.UpdateAlertClassesConfig(ctx, &client.AlertClassesUpdateRequest{
		Classes: map[string]client.AlertClassEntry{},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Resetting Alert Classes", fmt.Sprintf("Could not reset alert classes: %s", err))
		return
	}
	tflog.Trace(ctx, "Delete AlertClasses success")
}

func (r *AlertClassesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AlertClassesResource) buildUpdateRequest(ctx context.Context, plan *AlertClassesResourceModel) *client.AlertClassesUpdateRequest {
	out := map[string]client.AlertClassEntry{}

	if plan.Classes.IsNull() || plan.Classes.IsUnknown() {
		return &client.AlertClassesUpdateRequest{Classes: out}
	}

	// ElementsAs cannot fail: schema guarantees the map element type matches
	// AlertClassEntryModel, otherwise Plan.Get would have errored earlier.
	raw := map[string]AlertClassEntryModel{}
	_ = plan.Classes.ElementsAs(ctx, &raw, false)

	for k, v := range raw {
		entry := client.AlertClassEntry{}
		if !v.Level.IsNull() && !v.Level.IsUnknown() {
			entry.Level = v.Level.ValueString()
		}
		if !v.Policy.IsNull() && !v.Policy.IsUnknown() {
			entry.Policy = v.Policy.ValueString()
		}
		if !v.ProactiveSupport.IsNull() && !v.ProactiveSupport.IsUnknown() {
			b := v.ProactiveSupport.ValueBool()
			entry.ProactiveSupport = &b
		}
		out[k] = entry
	}

	return &client.AlertClassesUpdateRequest{Classes: out}
}

func (r *AlertClassesResource) mapResponseToModel(ctx context.Context, cfg *client.AlertClassesConfig, model *AlertClassesResourceModel, d interface{}) {
	model.ID = types.StringValue("alertclasses")

	objs := map[string]attr.Value{}
	for k, v := range cfg.Classes {
		level := types.StringNull()
		if v.Level != "" {
			level = types.StringValue(v.Level)
		}
		policy := types.StringNull()
		if v.Policy != "" {
			policy = types.StringValue(v.Policy)
		}
		proactive := types.BoolNull()
		if v.ProactiveSupport != nil {
			proactive = types.BoolValue(*v.ProactiveSupport)
		}
		// types.ObjectValue cannot fail with a fixed schema and matching values.
		obj, _ := types.ObjectValue(alertClassEntryAttrTypes, map[string]attr.Value{
			"level":             level,
			"policy":            policy,
			"proactive_support": proactive,
		})
		objs[k] = obj
	}

	mapVal, diags := types.MapValue(types.ObjectType{AttrTypes: alertClassEntryAttrTypes}, objs)
	if !diags.HasError() {
		model.Classes = mapVal
	}
	_ = ctx
	_ = d
}
