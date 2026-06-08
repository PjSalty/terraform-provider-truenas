package resources

// Filesystem ACL resource — manages NFS4/POSIX1E ACLs on TrueNAS SCALE
// datasets. ACLs are applied via POST /filesystem/setacl and read via
// POST /filesystem/getacl. See internal/client/filesystem_acl.go.

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &FilesystemACLResource{}
	_ resource.ResourceWithImportState = &FilesystemACLResource{}
)

// FilesystemACLResource manages filesystem ACLs on TrueNAS.
type FilesystemACLResource struct {
	client *client.Client
}

// FilesystemACLResourceModel describes the resource data model.
type FilesystemACLResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Path     types.String `tfsdk:"path"`
	ACLType  types.String `tfsdk:"acltype"`
	UID      types.Int64  `tfsdk:"uid"`
	GID      types.Int64  `tfsdk:"gid"`
	DACL     types.List   `tfsdk:"dacl"`
	Timeouts timeouts.

		// ACLEntryModel represents a single ACL entry in the Terraform model.
		Value `tfsdk:"timeouts"`
}

var aclEntryAttrTypes = map[string]attr.Type{
	"tag":          types.StringType,
	"id":           types.Int64Type,
	"perm_read":    types.BoolType,
	"perm_write":   types.BoolType,
	"perm_execute": types.BoolType,
	"default":      types.BoolType,
}

func NewFilesystemACLResource() resource.Resource {
	return &FilesystemACLResource{}
}

func (r *FilesystemACLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filesystem_acl"
}

func (r *FilesystemACLResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages POSIX ACLs on TrueNAS SCALE filesystem paths.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource identifier (same as path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The filesystem path to manage ACLs on (e.g., /mnt/pool/dataset).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^/`),
						"must be an absolute path",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"acltype": schema.StringAttribute{
				Description: "The ACL type (POSIX1E or NFS4).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("POSIX1E"),
				Validators: []validator.String{
					stringvalidator.OneOf("POSIX1E", "NFS4"),
				},
			},
			"uid": schema.Int64Attribute{
				Description: "The owner UID.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.Between(-1, 4294967295),
				},
			},
			"gid": schema.Int64Attribute{
				Description: "The owner GID.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.Between(-1, 4294967295),
				},
			},
			"dacl": schema.ListNestedAttribute{
				Description: "List of ACL entries.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"tag": schema.StringAttribute{
							Description: "ACL entry tag. POSIX1E: USER_OBJ, GROUP_OBJ, OTHER, USER, GROUP, MASK. NFS4: owner@, group@, everyone@, USER, GROUP.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									// POSIX1E tags
									"USER_OBJ", "GROUP_OBJ", "OTHER", "MASK",
									// NFS4 special principals
									"owner@", "group@", "everyone@",
									// Shared by both ACL types
									"USER", "GROUP",
								),
							},
						},
						"id": schema.Int64Attribute{
							Description: "The numeric user/group ID. Use -1 for USER_OBJ, GROUP_OBJ, OTHER, MASK.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(-1),
							Validators: []validator.Int64{
								int64validator.Between(-1, 4294967295),
							},
						},
						"perm_read": schema.BoolAttribute{
							Description: "Read permission.",
							Required:    true,
						},
						"perm_write": schema.BoolAttribute{
							Description: "Write permission.",
							Required:    true,
						},
						"perm_execute": schema.BoolAttribute{
							Description: "Execute permission.",
							Required:    true,
						},
						"default": schema.BoolAttribute{
							Description: "Whether this is a default ACL entry.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *FilesystemACLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FilesystemACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create FilesystemACL start")

	var plan FilesystemACLResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Setting filesystem ACL", map[string]interface{}{
		"path": plan.Path.ValueString(),
	})

	setReq, d := r.buildSetRequest(ctx, &plan)
	resp.Diagnostics.Append(d...)

	err := r.client.SetFilesystemACL(ctx, setReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Setting Filesystem ACL",
			fmt.Sprintf("Could not set ACL on %q: %s", plan.Path.ValueString(), err),
		)
		return
	}

	// Read back the ACL to populate computed fields
	acl, err := r.client.GetFilesystemACL(ctx, plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Filesystem ACL",
			fmt.Sprintf("Could not read ACL for %q after create: %s", plan.Path.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, acl, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create FilesystemACL success")
}

func (r *FilesystemACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read FilesystemACL start")

	var state FilesystemACLResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	acl, err := r.client.GetFilesystemACL(ctx, state.Path.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Filesystem ACL",
			fmt.Sprintf("Could not read ACL for %q: %s", state.Path.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, acl, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read FilesystemACL success")
}

func (r *FilesystemACLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update FilesystemACL start")

	var plan FilesystemACLResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setReq, d := r.buildSetRequest(ctx, &plan)
	resp.Diagnostics.Append(d...)

	err := r.client.SetFilesystemACL(ctx, setReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Filesystem ACL",
			fmt.Sprintf("Could not update ACL on %q: %s", plan.Path.ValueString(), err),
		)
		return
	}

	// Read back the ACL
	acl, err := r.client.GetFilesystemACL(ctx, plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Filesystem ACL",
			fmt.Sprintf("Could not read ACL for %q after update: %s", plan.Path.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, acl, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update FilesystemACL success")
}

func (r *FilesystemACLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete FilesystemACL start")

	var state FilesystemACLResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Resetting filesystem ACL to trivial defaults", map[string]interface{}{
		"path": state.Path.ValueString(),
	})

	// Reset to trivial POSIX ACL (owner rwx, group rx, other rx)
	uid := 0
	gid := 0
	err := r.client.SetFilesystemACL(ctx, &client.SetACLRequest{
		Path:    state.Path.ValueString(),
		ACLType: "POSIX1E",
		UID:     &uid,
		GID:     &gid,
		DACL: []client.SetACLEntry{
			{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: true, Execute: true}, Default: false},
			{Tag: "GROUP_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: false, Execute: true}, Default: false},
			{Tag: "OTHER", ID: -1, Perms: client.ACLPerms{Read: true, Write: false, Execute: true}, Default: false},
		},
	})
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Filesystem ACL path already gone, removing from state", map[string]interface{}{"path": state.Path.ValueString()})
			return
		}
		resp.Diagnostics.AddError(
			"Error Resetting Filesystem ACL",
			fmt.Sprintf("Could not reset ACL on %q: %s", state.Path.ValueString(), err),
		)
		return
	}
	tflog.Trace(ctx, "Delete FilesystemACL success")
}

func (r *FilesystemACLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *FilesystemACLResource) buildSetRequest(ctx context.Context, plan *FilesystemACLResourceModel) (*client.SetACLRequest, diag.Diagnostics) {
	var d diag.Diagnostics

	setReq := &client.SetACLRequest{
		Path:    plan.Path.ValueString(),
		ACLType: plan.ACLType.ValueString(),
	}

	if !plan.UID.IsNull() && !plan.UID.IsUnknown() {
		v := int(plan.UID.ValueInt64())
		setReq.UID = &v
	}
	if !plan.GID.IsNull() && !plan.GID.IsUnknown() {
		v := int(plan.GID.ValueInt64())
		setReq.GID = &v
	}

	// ElementsAs cannot fail when the target matches the schema list element
	// type (caught earlier by Plan.Get), so we do not check the diagnostics.
	var entries []types.Object
	d.Append(plan.DACL.ElementsAs(ctx, &entries, false)...)

	for _, entry := range entries {
		attrs := entry.Attributes()
		setReq.DACL = append(setReq.DACL, client.SetACLEntry{
			Tag: attrs["tag"].(types.String).ValueString(),
			ID:  int(attrs["id"].(types.Int64).ValueInt64()),
			Perms: client.ACLPerms{
				Read:    attrs["perm_read"].(types.Bool).ValueBool(),
				Write:   attrs["perm_write"].(types.Bool).ValueBool(),
				Execute: attrs["perm_execute"].(types.Bool).ValueBool(),
			},
			Default: attrs["default"].(types.Bool).ValueBool(),
		})
	}

	return setReq, d
}

func (r *FilesystemACLResource) mapResponseToModel(_ context.Context, acl *client.FilesystemACL, model *FilesystemACLResourceModel, d *diag.Diagnostics) {
	model.ID = types.StringValue(acl.Path)
	model.Path = types.StringValue(acl.Path)
	model.ACLType = types.StringValue(acl.ACLType)
	model.UID = types.Int64Value(int64(acl.UID))
	model.GID = types.Int64Value(int64(acl.GID))

	entryObjects := make([]attr.Value, 0, len(acl.ACL))
	for _, entry := range acl.ACL {
		obj, diags := types.ObjectValue(aclEntryAttrTypes, map[string]attr.Value{
			"tag":          types.StringValue(entry.Tag),
			"id":           types.Int64Value(int64(entry.ID)),
			"perm_read":    types.BoolValue(entry.Perms.Read),
			"perm_write":   types.BoolValue(entry.Perms.Write),
			"perm_execute": types.BoolValue(entry.Perms.Execute),
			"default":      types.BoolValue(entry.Default),
		})
		d.Append(diags...)
		entryObjects = append(entryObjects, obj)
	}

	listVal, diags := types.ListValue(types.ObjectType{AttrTypes: aclEntryAttrTypes}, entryObjects)
	d.Append(diags...)
	if !d.HasError() {
		model.DACL = listVal
	}
}
