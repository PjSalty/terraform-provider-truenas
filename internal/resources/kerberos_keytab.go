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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &KerberosKeytabResource{}
	_ resource.ResourceWithImportState = &KerberosKeytabResource{}
)

// KerberosKeytabResource manages a Kerberos keytab entry on TrueNAS.
type KerberosKeytabResource struct {
	client *client.Client
}

type KerberosKeytabResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	File     types.String   `tfsdk:"file"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewKerberosKeytabResource() resource.Resource {
	return &KerberosKeytabResource{}
}

func (r *KerberosKeytabResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kerberos_keytab"
}

func (r *KerberosKeytabResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Kerberos keytab entry on TrueNAS. Uploaded keytabs are merged " +
			"into the system keytab at /etc/krb5.keytab." + "\n\n" +
			"**Stability: GA.** Full `_basic` + `_disappears` + `_update` acceptance test triad verified live against TrueNAS SCALE 25.10 with a minimal MIT-format keytab fixture. Full integration with a real KDC has not been observed — the keytab is accepted and persisted correctly but never used for authentication during acc tests.",
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
				Description: "Numeric keytab ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Identifier for this keytab entry (e.g. SERVICE_PRINCIPAL). Note " +
					"that names like AD_MACHINE_ACCOUNT and IPA_MACHINE_ACCOUNT are reserved.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"file": schema.StringAttribute{
				Description: "Base64-encoded keytab file contents.",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *KerberosKeytabResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KerberosKeytabResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create KerberosKeytab start")

	var plan KerberosKeytabResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.KerberosKeytabCreateRequest{
		Name: plan.Name.ValueString(),
		File: plan.File.ValueString(),
	}

	tflog.Debug(ctx, "Creating kerberos keytab", map[string]interface{}{"name": createReq.Name})

	keytab, err := r.client.CreateKerberosKeytab(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Kerberos Keytab",
			fmt.Sprintf("Could not create kerberos keytab %q: %s", createReq.Name, err),
		)
		return
	}

	r.mapResponseToModel(keytab, &plan)
	// Preserve plan value for file — API may return normalized/re-encoded bytes.
	plan.File = types.StringValue(createReq.File)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create KerberosKeytab success")
}

func (r *KerberosKeytabResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read KerberosKeytab start")

	var state KerberosKeytabResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keytab ID: %s", err))
		return
	}

	keytab, err := r.client.GetKerberosKeytab(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Kerberos Keytab",
			fmt.Sprintf("Could not read kerberos keytab %d: %s", id, err),
		)
		return
	}

	// Only update mutable non-file fields. file is sensitive and API may
	// normalize whitespace; retaining plan value avoids spurious diffs.
	state.ID = types.StringValue(strconv.Itoa(keytab.ID))
	state.Name = types.StringValue(keytab.Name)
	if state.File.IsNull() && keytab.File != "" {
		state.File = types.StringValue(keytab.File)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read KerberosKeytab success")
}

func (r *KerberosKeytabResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update KerberosKeytab start")

	var plan KerberosKeytabResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state KerberosKeytabResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keytab ID: %s", err))
		return
	}

	name := plan.Name.ValueString()
	file := plan.File.ValueString()
	updateReq := &client.KerberosKeytabUpdateRequest{
		Name: &name,
		File: &file,
	}

	keytab, err := r.client.UpdateKerberosKeytab(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Kerberos Keytab",
			fmt.Sprintf("Could not update kerberos keytab %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(keytab, &plan)
	plan.File = types.StringValue(file)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update KerberosKeytab success")
}

func (r *KerberosKeytabResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete KerberosKeytab start")

	var state KerberosKeytabResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse keytab ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting kerberos keytab", map[string]interface{}{"id": id})
	if err := r.client.DeleteKerberosKeytab(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Kerberos keytab already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Kerberos Keytab",
			fmt.Sprintf("Could not delete kerberos keytab %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete KerberosKeytab success")
}

func (r *KerberosKeytabResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Kerberos keytab ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *KerberosKeytabResource) mapResponseToModel(keytab *client.KerberosKeytab, model *KerberosKeytabResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(keytab.ID))
	model.Name = types.StringValue(keytab.Name)
}
