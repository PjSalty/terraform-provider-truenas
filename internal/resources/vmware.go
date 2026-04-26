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
	_ resource.Resource                = &VMwareResource{}
	_ resource.ResourceWithImportState = &VMwareResource{}
)

// VMwareResource manages a VMware host registration on TrueNAS SCALE.
// It is used for snapshot-aware replication of VMware datastores that are
// backed by TrueNAS ZFS datasets.
type VMwareResource struct {
	client *client.Client
}

// VMwareResourceModel describes the resource data model.
type VMwareResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Datastore  types.String   `tfsdk:"datastore"`
	Filesystem types.String   `tfsdk:"filesystem"`
	Hostname   types.String   `tfsdk:"hostname"`
	Username   types.String   `tfsdk:"username"`
	Password   types.String   `tfsdk:"password"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func NewVMwareResource() resource.Resource {
	return &VMwareResource{}
}

func (r *VMwareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vmware"
}

func (r *VMwareResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VMware host registration on TrueNAS SCALE for snapshot-aware replication." + "\n\n" +
			"**Stability: Alpha.** Not end-to-end verified — requires a real vCenter instance. Schema matches the TrueNAS REST API.",
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
				Description: "The numeric ID of the VMware registration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"datastore": schema.StringAttribute{
				Description: "Datastore name that exists on the VMware host.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"filesystem": schema.StringAttribute{
				Description: "ZFS filesystem or dataset used for VMware storage.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "VMware host (or vCenter) hostname or IP address.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 253),
				},
			},
			"username": schema.StringAttribute{
				Description: "Username used to authenticate to the VMware host.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"password": schema.StringAttribute{
				Description: "Password used to authenticate to the VMware host.",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
				},
			},
		},
	}
}

func (r *VMwareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VMwareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create VMware start")

	var plan VMwareResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.VMwareCreateRequest{
		Datastore:  plan.Datastore.ValueString(),
		Filesystem: plan.Filesystem.ValueString(),
		Hostname:   plan.Hostname.ValueString(),
		Username:   plan.Username.ValueString(),
		Password:   plan.Password.ValueString(),
	}

	tflog.Debug(ctx, "Creating VMware registration", map[string]interface{}{
		"hostname": plan.Hostname.ValueString(),
	})

	v, err := r.client.CreateVMware(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VMware", fmt.Sprintf("Could not create VMware: %s", err))
		return
	}

	r.mapResponseToModel(v, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create VMware success")
}

func (r *VMwareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read VMware start")

	var state VMwareResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VMware ID: %s", err))
		return
	}

	v, err := r.client.GetVMware(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VMware", fmt.Sprintf("Could not read VMware %d: %s", id, err))
		return
	}

	r.mapResponseToModel(v, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read VMware success")
}

func (r *VMwareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update VMware start")

	var plan VMwareResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state VMwareResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VMware ID: %s", err))
		return
	}

	datastore := plan.Datastore.ValueString()
	filesystem := plan.Filesystem.ValueString()
	hostname := plan.Hostname.ValueString()
	username := plan.Username.ValueString()
	password := plan.Password.ValueString()

	updateReq := &client.VMwareUpdateRequest{
		Datastore:  &datastore,
		Filesystem: &filesystem,
		Hostname:   &hostname,
		Username:   &username,
		Password:   &password,
	}

	v, err := r.client.UpdateVMware(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating VMware", fmt.Sprintf("Could not update VMware %d: %s", id, err))
		return
	}

	r.mapResponseToModel(v, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update VMware success")
}

func (r *VMwareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete VMware start")

	var state VMwareResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VMware ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting VMware", map[string]interface{}{"id": id})

	if err := r.client.DeleteVMware(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "VMware snapshot already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting VMware", fmt.Sprintf("Could not delete VMware %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete VMware success")
}

func (r *VMwareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("VMware ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), types.StringValue(""))...)
}

// mapResponseToModel copies API fields to the model, preserving the
// user-supplied password value (API returns it redacted).
func (r *VMwareResource) mapResponseToModel(v *client.VMware, model *VMwareResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(v.ID))
	model.Datastore = types.StringValue(v.Datastore)
	model.Filesystem = types.StringValue(v.Filesystem)
	model.Hostname = types.StringValue(v.Hostname)
	model.Username = types.StringValue(v.Username)
	// Preserve whatever password the user configured; API redacts this field.
	if model.Password.IsNull() || model.Password.IsUnknown() {
		model.Password = types.StringValue("")
	}
}
