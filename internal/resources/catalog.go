package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &CatalogResource{}
	_ resource.ResourceWithImportState = &CatalogResource{}
)

// CatalogResource manages the TrueNAS SCALE application catalog.
//
// In TrueNAS SCALE 25.04+ the catalog is a singleton — there is exactly one
// catalog (label "TRUENAS") and only `preferred_trains` is user-tunable via
// the REST API. This resource therefore behaves like the `ssh_config`
// singleton: Create/Update/Read all hit PUT/GET /catalog, and Delete resets
// the preferred trains to an empty list.
type CatalogResource struct {
	client *client.Client
}

// CatalogResourceModel describes the resource data model.
type CatalogResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	Label           types.String   `tfsdk:"label"`
	PreferredTrains types.List     `tfsdk:"preferred_trains"`
	Location        types.String   `tfsdk:"location"`
	SyncOnCreate    types.Bool     `tfsdk:"sync_on_create"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

func NewCatalogResource() resource.Resource {
	return &CatalogResource{}
}

func (r *CatalogResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog"
}

func (r *CatalogResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the TrueNAS SCALE application catalog. This is a singleton " +
			"resource in SCALE 25.04+ — only `preferred_trains` is user-tunable. " +
			"Optionally triggers a catalog sync on create.",
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
				Description: "The catalog ID as reported by TrueNAS.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				Description: "The catalog label (always 'TRUENAS' in SCALE 25.04+).",
				Computed:    true,
			},
			"preferred_trains": schema.ListAttribute{
				Description: "Trains to prefer when searching for app versions " +
					"(e.g. ['stable'], ['stable', 'community']).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("stable"),
				})),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"location": schema.StringAttribute{
				Description: "Local filesystem path where the catalog is checked out.",
				Computed:    true,
			},
			"sync_on_create": schema.BoolAttribute{
				Description: "If true, trigger a catalog sync job after updating " +
					"preferred_trains on create (default false).",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *CatalogResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CatalogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Catalog start")

	var plan CatalogResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	trains, d := listToStringSlice(ctx, plan.PreferredTrains)
	resp.Diagnostics.Append(d...)

	updateReq := &client.CatalogUpdateRequest{PreferredTrains: &trains}
	tflog.Debug(ctx, "Updating TrueNAS catalog (singleton create)", map[string]interface{}{
		"preferred_trains": trains,
	})

	cat, err := r.client.UpdateCatalog(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Catalog",
			fmt.Sprintf("Could not update catalog: %s", err),
		)
		return
	}

	if plan.SyncOnCreate.ValueBool() {
		tflog.Debug(ctx, "Triggering catalog sync")
		if err := r.client.SyncCatalog(ctx); err != nil {
			resp.Diagnostics.AddError(
				"Error Syncing Catalog",
				fmt.Sprintf("Could not sync catalog: %s", err),
			)
			return
		}
	}

	r.mapResponseToModel(ctx, cat, &plan)
	if plan.SyncOnCreate.IsNull() || plan.SyncOnCreate.IsUnknown() {
		plan.SyncOnCreate = types.BoolValue(false)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Catalog success")
}

func (r *CatalogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Catalog start")

	var state CatalogResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cat, err := r.client.GetCatalog(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Catalog",
			fmt.Sprintf("Could not read catalog: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, cat, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Catalog success")
}

func (r *CatalogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Catalog start")

	var plan CatalogResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	trains, d := listToStringSlice(ctx, plan.PreferredTrains)
	resp.Diagnostics.Append(d...)

	cat, err := r.client.UpdateCatalog(ctx, &client.CatalogUpdateRequest{PreferredTrains: &trains})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Catalog",
			fmt.Sprintf("Could not update catalog: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, cat, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Catalog success")
}

func (r *CatalogResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Catalog start")

	// Singleton — cannot be deleted. Reset preferred_trains to the SCALE
	// default ["stable"] so the catalog is in a known state for the next
	// apply.
	tflog.Debug(ctx, "Resetting TrueNAS catalog preferred_trains to default on delete")

	trains := []string{"stable"}
	if _, err := r.client.UpdateCatalog(ctx, &client.CatalogUpdateRequest{PreferredTrains: &trains}); err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting Catalog",
			fmt.Sprintf("Could not reset catalog preferred_trains: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Catalog success")
}

func (r *CatalogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sync_on_create"), types.BoolValue(false))...)
}

func (r *CatalogResource) mapResponseToModel(ctx context.Context, cat *client.Catalog, model *CatalogResourceModel) {
	model.ID = types.StringValue(cat.ID)
	model.Label = types.StringValue(cat.Label)
	model.Location = types.StringValue(cat.Location)

	trainValues, diags := types.ListValueFrom(ctx, types.StringType, cat.PreferredTrains)
	if !diags.HasError() {
		model.PreferredTrains = trainValues
	}
}

// listToStringSlice extracts a Go []string from a types.List of strings.
func listToStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	out := []string{}
	if list.IsNull() || list.IsUnknown() {
		return out, nil
	}
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}
