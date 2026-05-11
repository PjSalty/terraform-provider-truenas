// reporting_exporter.go
//
// Uses an `attributes_json` string because the `attributes` payload is
// polymorphic (discriminated by `exporter_type`) — e.g. Graphite uses
// destination_ip/destination_port/namespace while other types may differ.
// Users pass `attributes_json = jsonencode({ exporter_type = "GRAPHITE", ... })`.
package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &ReportingExporterResource{}
	_ resource.ResourceWithImportState = &ReportingExporterResource{}
)

type ReportingExporterResource struct {
	client *client.Client
}

type ReportingExporterResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Enabled        types.Bool     `tfsdk:"enabled"`
	AttributesJSON types.String   `tfsdk:"attributes_json"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func NewReportingExporterResource() resource.Resource {
	return &ReportingExporterResource{}
}

func (r *ReportingExporterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reporting_exporter"
}

func (r *ReportingExporterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a reporting exporter (e.g. Graphite) on TrueNAS SCALE.",
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
			"name": schema.StringAttribute{
				Description: "User-defined name of the exporter.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether this exporter is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"attributes_json": schema.StringAttribute{
				Description: "Exporter-specific attributes as a JSON object, including `exporter_type`. " +
					"Example: jsonencode({exporter_type=\"GRAPHITE\", destination_ip=\"1.2.3.4\", destination_port=2003, namespace=\"truenas\"}). " +
					"May contain credentials (passwords, tokens) for remote exporters — marked sensitive.",
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func (r *ReportingExporterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReportingExporterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ReportingExporter start")

	var plan ReportingExporterResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs, err := normalizeJSON(plan.AttributesJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid attributes_json", err.Error())
		return
	}

	createReq := &client.ReportingExporterCreateRequest{
		Name:       plan.Name.ValueString(),
		Enabled:    plan.Enabled.ValueBool(),
		Attributes: attrs,
	}

	tflog.Debug(ctx, "Creating reporting exporter", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	e, err := r.client.CreateReportingExporter(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Reporting Exporter", fmt.Sprintf("Could not create exporter: %s", err))
		return
	}

	r.mapResponseToModel(e, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ReportingExporter success")
}

func (r *ReportingExporterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ReportingExporter start")

	var state ReportingExporterResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse reporting exporter ID: %s", err))
		return
	}

	e, err := r.client.GetReportingExporter(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Reporting Exporter", fmt.Sprintf("Could not read exporter %d: %s", id, err))
		return
	}

	r.mapResponseToModel(e, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ReportingExporter success")
}

func (r *ReportingExporterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ReportingExporter start")

	var plan ReportingExporterResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ReportingExporterResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse reporting exporter ID: %s", err))
		return
	}

	attrs, err := normalizeJSON(plan.AttributesJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid attributes_json", err.Error())
		return
	}

	enabled := plan.Enabled.ValueBool()
	name := plan.Name.ValueString()

	updateReq := &client.ReportingExporterUpdateRequest{
		Enabled:    &enabled,
		Name:       &name,
		Attributes: attrs,
	}

	e, err := r.client.UpdateReportingExporter(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Reporting Exporter", fmt.Sprintf("Could not update exporter %d: %s", id, err))
		return
	}

	r.mapResponseToModel(e, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ReportingExporter success")
}

func (r *ReportingExporterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ReportingExporter start")

	var state ReportingExporterResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse reporting exporter ID: %s", err))
		return
	}

	if err := r.client.DeleteReportingExporter(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Reporting exporter already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Reporting Exporter", fmt.Sprintf("Could not delete exporter %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete ReportingExporter success")
}

func (r *ReportingExporterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Reporting exporter ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ReportingExporterResource) mapResponseToModel(e *client.ReportingExporter, model *ReportingExporterResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(e.ID))
	model.Name = types.StringValue(e.Name)
	model.Enabled = types.BoolValue(e.Enabled)
	if len(e.Attributes) > 0 {
		// Preserve the user-supplied subset of attributes. TrueNAS fills in
		// defaults server-side (matching_charts, send_names_instead_of_ids, ...)
		// that the user never wrote — if we stored the full response, every
		// subsequent plan would show spurious drift.
		prior := model.AttributesJSON.ValueString()
		if filtered, err := filterJSONByKeys(string(e.Attributes), prior); err == nil {
			// filtered is re-marshaled JSON; normalizeJSON cannot fail.
			canon, _ := normalizeJSON(filtered)
			model.AttributesJSON = types.StringValue(string(canon))
		} else {
			// filterJSONByKeys only errors on malformed e.Attributes.
			model.AttributesJSON = types.StringValue(string(e.Attributes))
		}
	}
}
