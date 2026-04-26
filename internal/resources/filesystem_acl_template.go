// filesystem_acl_template.go
//
// Uses an `acl_json` string for the ACL payload because ACE entries are
// polymorphic between NFS4 and POSIX1E templates — NFS4 uses a full
// perms/flags object, POSIX1E uses a simpler perm string. Mashing both
// shapes into one nested block would be fragile, so we accept a JSON
// list instead: `acl_json = jsonencode([{tag = "owner@", type = "ALLOW", ...}])`.
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &FilesystemACLTemplateResource{}
	_ resource.ResourceWithImportState = &FilesystemACLTemplateResource{}
)

type FilesystemACLTemplateResource struct {
	client *client.Client
}

type FilesystemACLTemplateResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	ACLType  types.String   `tfsdk:"acltype"`
	Comment  types.String   `tfsdk:"comment"`
	ACLJSON  types.String   `tfsdk:"acl_json"`
	Builtin  types.Bool     `tfsdk:"builtin"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewFilesystemACLTemplateResource() resource.Resource {
	return &FilesystemACLTemplateResource{}
}

func (r *FilesystemACLTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filesystem_acl_template"
}

func (r *FilesystemACLTemplateResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a filesystem ACL template on TrueNAS SCALE.",
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
				Description: "Human-readable template name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"acltype": schema.StringAttribute{
				Description: "ACL type this template provides: NFS4 or POSIX1E.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("NFS4", "POSIX1E"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comment": schema.StringAttribute{
				Description: "Optional comment.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"acl_json": schema.StringAttribute{
				Description: "ACL entries as a JSON array (see resource docs).",
				Required:    true,
			},
			"builtin": schema.BoolAttribute{
				Description: "True if this is a TrueNAS built-in template (read-only).",
				Computed:    true,
			},
		},
	}
}

func (r *FilesystemACLTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FilesystemACLTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create FilesystemACLTemplate start")

	var plan FilesystemACLTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	acl, err := normalizeJSON(plan.ACLJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid acl_json", err.Error())
		return
	}

	createReq := &client.FilesystemACLTemplateCreateRequest{
		Name:    plan.Name.ValueString(),
		ACLType: plan.ACLType.ValueString(),
		Comment: plan.Comment.ValueString(),
		ACL:     acl,
	}

	tflog.Debug(ctx, "Creating filesystem ACL template", map[string]interface{}{"name": createReq.Name})

	t, err := r.client.CreateFilesystemACLTemplate(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating ACL Template", fmt.Sprintf("Could not create ACL template: %s", err))
		return
	}

	r.mapResponseToModel(t, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create FilesystemACLTemplate success")
}

func (r *FilesystemACLTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read FilesystemACLTemplate start")

	var state FilesystemACLTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse ACL template ID: %s", err))
		return
	}

	t, err := r.client.GetFilesystemACLTemplate(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading ACL Template", fmt.Sprintf("Could not read ACL template %d: %s", id, err))
		return
	}

	r.mapResponseToModel(t, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read FilesystemACLTemplate success")
}

func (r *FilesystemACLTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update FilesystemACLTemplate start")

	var plan FilesystemACLTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FilesystemACLTemplateResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse ACL template ID: %s", err))
		return
	}

	acl, err := normalizeJSON(plan.ACLJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid acl_json", err.Error())
		return
	}

	name := plan.Name.ValueString()
	comment := plan.Comment.ValueString()

	updateReq := &client.FilesystemACLTemplateUpdateRequest{
		Name:    &name,
		Comment: &comment,
		ACL:     acl,
	}

	t, err := r.client.UpdateFilesystemACLTemplate(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating ACL Template", fmt.Sprintf("Could not update ACL template %d: %s", id, err))
		return
	}

	r.mapResponseToModel(t, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update FilesystemACLTemplate success")
}

func (r *FilesystemACLTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete FilesystemACLTemplate start")

	var state FilesystemACLTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse ACL template ID: %s", err))
		return
	}

	if err := r.client.DeleteFilesystemACLTemplate(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Filesystem ACL template already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting ACL Template", fmt.Sprintf("Could not delete ACL template %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete FilesystemACLTemplate success")
}

func (r *FilesystemACLTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("ACL template ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *FilesystemACLTemplateResource) mapResponseToModel(t *client.FilesystemACLTemplate, model *FilesystemACLTemplateResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(t.ID))
	model.Name = types.StringValue(t.Name)
	model.ACLType = types.StringValue(t.ACLType)
	model.Comment = types.StringValue(t.Comment)
	model.Builtin = types.BoolValue(t.Builtin)
	if len(t.ACL) > 0 {
		// Strip server-added null fields (e.g. "who": null) before storing,
		// so Create/Update round-trips match the user's plan and no drift
		// appears on subsequent refreshes.
		if stripped, err := stripJSONNulls(string(t.ACL)); err == nil {
			// normalizeJSON cannot fail on output produced by stripJSONNulls.
			canon, _ := normalizeJSON(stripped)
			model.ACLJSON = types.StringValue(string(canon))
		} else {
			model.ACLJSON = types.StringValue(string(t.ACL))
		}
	}
}
