package resources

// Directory resource, manages a directory on a TrueNAS SCALE filesystem
// via filesystem.mkdir / filesystem.stat / filesystem.setperm. The
// resource is path-keyed (id == path) like filesystem_acl. TrueNAS
// exposes no directory-removal method, so Delete is state-only.

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                = &DirectoryResource{}
	_ resource.ResourceWithImportState = &DirectoryResource{}
)

// DirectoryResource manages a directory on TrueNAS.
type DirectoryResource struct {
	client *wsclient.Client
}

// DirectoryResourceModel describes the resource data model.
type DirectoryResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Path          types.String   `tfsdk:"path"`
	Mode          types.String   `tfsdk:"mode"`
	CreateParents types.Bool     `tfsdk:"create_parents"`
	UID           types.Int64    `tfsdk:"uid"`
	GID           types.Int64    `tfsdk:"gid"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewDirectoryResource() resource.Resource {
	return &DirectoryResource{}
}

func (r *DirectoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

func (r *DirectoryResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a directory on a TrueNAS SCALE filesystem path.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource identifier (same as path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The absolute filesystem path of the directory (e.g., /mnt/tank/media). Must be under /mnt/.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^/mnt/`),
						"path must be under /mnt/",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "The octal permission mode for the directory (e.g., 755).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("755"),
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-7]{3,4}$`),
						"mode must be 3 or 4 octal digits (e.g., 755 or 0755)",
					),
				},
			},
			"create_parents": schema.BoolAttribute{
				Description: "When true, create any missing parent directories before the leaf (like mkdir -p).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"uid": schema.Int64Attribute{
				Description: "The owner UID. Applied via setperm when set or changed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(0, 4294967295),
				},
			},
			"gid": schema.Int64Attribute{
				Description: "The owner GID. Applied via setperm when set or changed.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(0, 4294967295),
				},
			},
		},
	}
}

func (r *DirectoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Directory start")

	var plan DirectoryResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dirPath := plan.Path.ValueString()
	mode := plan.Mode.ValueString()

	// create_parents walks the absolute path and mkdirs missing
	// ancestors client-side; the API mkdir has no parents option.
	if plan.CreateParents.ValueBool() {
		if err := r.mkdirParents(ctx, dirPath, mode); err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Directory",
				fmt.Sprintf("Could not create parent directories for %q: %s", dirPath, err),
			)
			return
		}
	}

	raise := false
	if _, err := r.client.Mkdir(ctx, &truenas.MkdirRequest{
		Path: dirPath,
		Options: &truenas.MkdirOptions{
			Mode:            mode,
			RaiseChmodError: &raise,
		},
	}); err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Directory",
			fmt.Sprintf("Could not create directory %q: %s", dirPath, err),
		)
		return
	}

	if !plan.UID.IsNull() && !plan.UID.IsUnknown() || !plan.GID.IsNull() && !plan.GID.IsUnknown() {
		if err := r.applyPerm(ctx, &plan); err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Directory",
				fmt.Sprintf("Could not set perms on %q: %s", dirPath, err),
			)
			return
		}
	}

	stat, err := r.client.StatFilesystem(ctx, dirPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Directory",
			fmt.Sprintf("Could not stat %q after create: %s", dirPath, err),
		)
		return
	}

	// setperm completed SUCCESS; trust our applied uid/gid over
	// potentially stale stat data (TrueNAS caching).
	uid := plan.UID
	gid := plan.GID
	r.mapStatToModel(stat, &plan)
	if !uid.IsNull() && !uid.IsUnknown() {
		plan.UID = uid
	}
	if !gid.IsNull() && !gid.IsUnknown() {
		plan.GID = gid
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Directory success")
}

func (r *DirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Directory start")

	var state DirectoryResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stat, err := r.client.StatFilesystem(ctx, state.Path.ValueString())
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Directory",
			fmt.Sprintf("Could not stat %q: %s", state.Path.ValueString(), err),
		)
		return
	}

	r.mapStatToModel(stat, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Directory success")
}

func (r *DirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Directory start")

	var plan DirectoryResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// path is RequiresReplace, so only mode/uid/gid can change in place.
	// setperm carries all three in one call.
	if err := r.applyPerm(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Directory",
			fmt.Sprintf("Could not update perms on %q: %s", plan.Path.ValueString(), err),
		)
		return
	}

	stat, err := r.client.StatFilesystem(ctx, plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Directory",
			fmt.Sprintf("Could not stat %q after update: %s", plan.Path.ValueString(), err),
		)
		return
	}

	// setperm completed SUCCESS; trust our applied uid/gid over
	// potentially stale stat data (TrueNAS caching).
	uid := plan.UID
	gid := plan.GID
	r.mapStatToModel(stat, &plan)
	if !uid.IsNull() && !uid.IsUnknown() {
		plan.UID = uid
	}
	if !gid.IsNull() && !gid.IsUnknown() {
		plan.GID = gid
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Directory success")
}

func (r *DirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Directory start")

	var state DirectoryResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TrueNAS exposes no directory-removal method (no rmdir/unlink), so
	// Delete is state-only. If the path is already gone, IsNotFound
	// lets us return cleanly; otherwise we warn that the directory is
	// left on disk and return without error so destroy stays idempotent.
	dirPath := state.Path.ValueString()
	_, err := r.client.StatFilesystem(ctx, dirPath)
	if err != nil && wsclient.IsNotFound(err) {
		return
	}
	tflog.Warn(ctx, "directory left on disk: TrueNAS exposes no directory-removal API", map[string]interface{}{
		"path": dirPath,
	})
	tflog.Trace(ctx, "Delete Directory success")
}

// ImportState seeds both id and path from the import ID (the absolute
// path). Without seeding path, the next Read would stat an empty path.
// Mirrors the filesystem_acl import pattern.
func (r *DirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
}

// mkdirParents stats each ancestor of dirPath and mkdirs the ones that
// are missing, leaf-exclusive. Client-side because the API mkdir has no
// parents option.
func (r *DirectoryResource) mkdirParents(ctx context.Context, dirPath, mode string) error {
	clean := strings.TrimRight(dirPath, "/")
	idx := strings.LastIndex(clean, "/")
	if idx <= 0 {
		return nil
	}
	parent := clean[:idx]

	// Build the ancestor chain from shallowest to deepest, stopping at
	// /mnt (the pool mount root, which always exists).
	segments := strings.Split(strings.TrimPrefix(parent, "/"), "/")
	cur := ""
	var ancestors []string
	for _, seg := range segments {
		cur += "/" + seg
		ancestors = append(ancestors, cur)
	}

	raise := false
	for _, anc := range ancestors {
		// /mnt and the pool root are server-managed; never mkdir them.
		if anc == "/mnt" {
			continue
		}
		_, err := r.client.StatFilesystem(ctx, anc)
		if err == nil {
			continue
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("stating ancestor %q: %w", anc, err)
		}
		if _, err := r.client.Mkdir(ctx, &truenas.MkdirRequest{
			Path: anc,
			Options: &truenas.MkdirOptions{
				Mode:            mode,
				RaiseChmodError: &raise,
			},
		}); err != nil {
			return fmt.Errorf("creating ancestor %q: %w", anc, err)
		}
	}
	return nil
}

// applyPerm calls setperm with the plan's mode/uid/gid. mode is always
// sent; uid/gid only when set.
func (r *DirectoryResource) applyPerm(ctx context.Context, plan *DirectoryResourceModel) error {
	mode := plan.Mode.ValueString()
	setReq := &truenas.SetPermRequest{
		Path: plan.Path.ValueString(),
		Mode: &mode,
	}
	if !plan.UID.IsNull() && !plan.UID.IsUnknown() {
		v := int(plan.UID.ValueInt64())
		setReq.UID = &v
	}
	if !plan.GID.IsNull() && !plan.GID.IsUnknown() {
		v := int(plan.GID.ValueInt64())
		setReq.GID = &v
	}
	return r.client.SetFilesystemPerm(ctx, setReq)
}

// mapStatToModel reconciles state from a stat result. id is keyed to
// the managed path (model.Path is the source of truth and stays as the
// user-supplied value so import and round-trip are stable). mode is
// masked to the perm bits and formatted as a 3-digit octal string;
// uid/gid come straight from the stat.
func (r *DirectoryResource) mapStatToModel(stat *truenas.FilesystemStat, model *DirectoryResourceModel) {
	model.ID = model.Path
	// keep the user's mode spelling (e.g. "0755" vs "755") when it matches
	// the on-disk perm bits, so equivalent octal forms don't churn the plan;
	// only overwrite on real drift or when mode is unset (import).
	statMode := stat.Mode & 0o7777
	if cur, err := strconv.ParseInt(model.Mode.ValueString(), 8, 32); err != nil || int(cur) != statMode {
		model.Mode = types.StringValue(fmt.Sprintf("%03o", statMode))
	}
	model.UID = types.Int64Value(int64(stat.UID))
	model.GID = types.Int64Value(int64(stat.GID))
}
