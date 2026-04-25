package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &KerberosRealmResource{}
	_ resource.ResourceWithImportState = &KerberosRealmResource{}
)

// KerberosRealmResource manages a Kerberos realm on TrueNAS.
type KerberosRealmResource struct {
	client *client.Client
}

type KerberosRealmResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Realm         types.String   `tfsdk:"realm"`
	PrimaryKDC    types.String   `tfsdk:"primary_kdc"`
	KDC           types.List     `tfsdk:"kdc"`
	AdminServer   types.List     `tfsdk:"admin_server"`
	KPasswdServer types.List     `tfsdk:"kpasswd_server"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewKerberosRealmResource() resource.Resource {
	return &KerberosRealmResource{}
}

func (r *KerberosRealmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kerberos_realm"
}

func (r *KerberosRealmResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyStringList := types.ListValueMust(types.StringType, nil)
	resp.Schema = schema.Schema{
		Description: "Manages a Kerberos realm on TrueNAS. Realm names are case-sensitive " +
			"and conventionally upper-case.",
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
				Description: "Numeric realm ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"realm": schema.StringAttribute{
				Description: "Kerberos realm name (e.g. EXAMPLE.COM).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"primary_kdc": schema.StringAttribute{
				Description: "Optional primary/master KDC hostname. Used as a fallback when the " +
					"machine password is invalid.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 255),
				},
			},
			"kdc": schema.ListAttribute{
				Description: "List of Kerberos KDC hostnames. If empty, libraries use DNS SRV lookups.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(emptyStringList),
			},
			"admin_server": schema.ListAttribute{
				Description: "List of Kerberos admin server hostnames.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(emptyStringList),
			},
			"kpasswd_server": schema.ListAttribute{
				Description: "List of Kerberos kpasswd server hostnames.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(emptyStringList),
			},
		},
	}
}

func (r *KerberosRealmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KerberosRealmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create KerberosRealm start")

	var plan KerberosRealmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kdc := privilegeListToStringSlice(ctx, plan.KDC, &resp.Diagnostics)
	admin := privilegeListToStringSlice(ctx, plan.AdminServer, &resp.Diagnostics)
	kpasswd := privilegeListToStringSlice(ctx, plan.KPasswdServer, &resp.Diagnostics)

	createReq := &client.KerberosRealmCreateRequest{
		Realm:         plan.Realm.ValueString(),
		KDC:           kdc,
		AdminServer:   admin,
		KPasswdServer: kpasswd,
	}
	if !plan.PrimaryKDC.IsNull() && !plan.PrimaryKDC.IsUnknown() && plan.PrimaryKDC.ValueString() != "" {
		v := plan.PrimaryKDC.ValueString()
		createReq.PrimaryKDC = &v
	}

	tflog.Debug(ctx, "Creating kerberos realm", map[string]interface{}{"realm": createReq.Realm})

	realm, err := r.client.CreateKerberosRealm(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Kerberos Realm",
			fmt.Sprintf("Could not create kerberos realm %q: %s", createReq.Realm, err),
		)
		return
	}

	r.mapResponseToModel(ctx, realm, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create KerberosRealm success")
}

func (r *KerberosRealmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read KerberosRealm start")

	var state KerberosRealmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse realm ID: %s", err))
		return
	}

	realm, err := r.client.GetKerberosRealm(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Kerberos Realm",
			fmt.Sprintf("Could not read kerberos realm %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, realm, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read KerberosRealm success")
}

func (r *KerberosRealmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update KerberosRealm start")

	var plan KerberosRealmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state KerberosRealmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse realm ID: %s", err))
		return
	}

	kdc := privilegeListToStringSlice(ctx, plan.KDC, &resp.Diagnostics)
	admin := privilegeListToStringSlice(ctx, plan.AdminServer, &resp.Diagnostics)
	kpasswd := privilegeListToStringSlice(ctx, plan.KPasswdServer, &resp.Diagnostics)

	realmName := plan.Realm.ValueString()
	updateReq := &client.KerberosRealmUpdateRequest{
		Realm:         &realmName,
		KDC:           &kdc,
		AdminServer:   &admin,
		KPasswdServer: &kpasswd,
	}
	if !plan.PrimaryKDC.IsNull() && !plan.PrimaryKDC.IsUnknown() && plan.PrimaryKDC.ValueString() != "" {
		v := plan.PrimaryKDC.ValueString()
		updateReq.PrimaryKDC = &v
	}

	realm, err := r.client.UpdateKerberosRealm(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Kerberos Realm",
			fmt.Sprintf("Could not update kerberos realm %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, realm, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update KerberosRealm success")
}

func (r *KerberosRealmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete KerberosRealm start")

	var state KerberosRealmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse realm ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting kerberos realm", map[string]interface{}{"id": id})
	if err := r.client.DeleteKerberosRealm(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Kerberos realm already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Kerberos Realm",
			fmt.Sprintf("Could not delete kerberos realm %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete KerberosRealm success")
}

func (r *KerberosRealmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Kerberos realm ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *KerberosRealmResource) mapResponseToModel(ctx context.Context, realm *client.KerberosRealm, model *KerberosRealmResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(strconv.Itoa(realm.ID))
	model.Realm = types.StringValue(realm.Realm)

	if realm.PrimaryKDC != nil {
		model.PrimaryKDC = types.StringValue(*realm.PrimaryKDC)
	} else {
		model.PrimaryKDC = types.StringNull()
	}

	kdcList, d := types.ListValueFrom(ctx, types.StringType, realm.KDC)
	diags.Append(d...)
	model.KDC = kdcList

	adminList, d := types.ListValueFrom(ctx, types.StringType, realm.AdminServer)
	diags.Append(d...)
	model.AdminServer = adminList

	kpasswdList, d := types.ListValueFrom(ctx, types.StringType, realm.KPasswdServer)
	diags.Append(d...)
	model.KPasswdServer = kpasswdList
}
