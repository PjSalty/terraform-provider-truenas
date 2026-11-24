package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// diskStandbyValues and diskAdvPowerMgmtValues are the literals of
// DiskEntry.hddstandby and DiskEntry.advpowermgmt in the 25.10 API model.
// TrueNAS stores them title-cased and reads them back upper-cased, so only
// the upper-case spelling is accepted: "Always On" would plan a change on
// every run.
var (
	diskStandbyValues = []string{
		"ALWAYS ON", "5", "10", "20", "30", "60", "120", "180", "240", "300", "330",
	}
	diskAdvPowerMgmtValues = []string{
		"DISABLED", "1", "64", "127", "128", "192", "254",
	}
)

var (
	_ resource.Resource                = &DiskResource{}
	_ resource.ResourceWithImportState = &DiskResource{}
)

// DiskResource manages the stored per-disk settings of a disk TrueNAS
// already knows about. It does not create or destroy anything: a disk is
// hardware.
type DiskResource struct {
	client *wsclient.Client
}

// DiskResourceModel describes the resource data model.
type DiskResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Identifier   types.String   `tfsdk:"identifier"`
	HDDStandby   types.String   `tfsdk:"hddstandby"`
	AdvPowerMgmt types.String   `tfsdk:"advpowermgmt"`
	Description  types.String   `tfsdk:"description"`
	Name         types.String   `tfsdk:"name"`
	Serial       types.String   `tfsdk:"serial"`
	Model        types.String   `tfsdk:"model"`
	Size         types.Int64    `tfsdk:"size"`
	Type         types.String   `tfsdk:"type"`
	Pool         types.String   `tfsdk:"pool"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

// NewDiskResource returns a new DiskResource factory.
func NewDiskResource() resource.Resource {
	return &DiskResource{}
}

func (r *DiskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_disk"
}

func (r *DiskResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the stored settings of one disk TrueNAS already knows about: its standby timer, " +
			"its Advanced Power Management level and its description. A disk is hardware, so this resource " +
			"never creates or destroys one. Creating it adopts the disk named by `identifier` and fails if " +
			"there is no such disk; destroying it only stops Terraform managing the disk and leaves every " +
			"setting as it is, and nothing is wiped. Changing a standby timer or power level makes TrueNAS " +
			"run `hdparm` against the drive in the background; a drive or controller that ignores the " +
			"command, such as a virtual disk, is not an error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The disk identifier, the same value as `identifier`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identifier": schema.StringAttribute{
				Description: "The TrueNAS disk identifier, for example `{serial_lunid}WD-WCC4N1234567_5000cca2` " +
					"(the `identifier` attribute of the `truenas_disk` data source, which finds a disk by " +
					"its device name). It is also the import ID. Unlike the device name, it follows the " +
					"drive across reboots. Changing it forces a new resource.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hddstandby": schema.StringAttribute{
				Description: "Minutes of inactivity before the drive spins down: one of `ALWAYS ON` (never), " +
					"`5`, `10`, `20`, `30`, `60`, `120`, `180`, `240`, `300` or `330`. Left out, the current " +
					"value is kept; TrueNAS defaults to `ALWAYS ON`. After a change TrueNAS waits a minute " +
					"and then runs `hdparm -S` against the drive.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(diskStandbyValues...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"advpowermgmt": schema.StringAttribute{
				Description: "Advanced Power Management level, sent to the drive with `hdparm -B`: one of " +
					"`DISABLED`, `1`, `64`, `127`, `128`, `192` or `254`. `1` to `127` allow the drive to " +
					"spin down, `128` and above do not, and a higher number favors performance. Left out, " +
					"the current value is kept; TrueNAS defaults to `DISABLED`.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(diskAdvPowerMgmtValues...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Free-text description of the disk, shown in the TrueNAS UI. Left out, the " +
					"current value is kept.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The current device name, for example `sda`. Read-only, and it is not stable: " +
					"Linux can name the same drive differently after a reboot, which is why `identifier` " +
					"names the disk. A new name shows up on the next refresh and is not a change to " +
					"configure.",
				Computed: true,
			},
			"serial": schema.StringAttribute{
				Description: "The disk serial number. Read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"model": schema.StringAttribute{
				Description: "The disk model. Empty when TrueNAS does not report one. Read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				Description: "The disk size in bytes. `0` when TrueNAS does not report one. Read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The disk type, `SSD` or `HDD`. Empty when TrueNAS has not determined one. Read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool": schema.StringAttribute{
				Description: "The name of the pool the disk is a member of, or null when it is in none. Read-only.",
				Computed:    true,
			},
		},
	}
}

func (r *DiskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// updateRequest is the request form of the plan. Only the settings the plan
// set are included: disk.update is a ForUpdateMetaclass model, so an
// omitted key leaves the stored value alone. The hardware attributes of the
// model (bus, number, lunid, pool, enclosure) and the SED password are never
// sent.
func (r *DiskResource) updateRequest(plan *DiskResourceModel) *truenas.DiskUpdateRequest {
	req := &truenas.DiskUpdateRequest{}
	if !plan.HDDStandby.IsNull() && !plan.HDDStandby.IsUnknown() {
		v := plan.HDDStandby.ValueString()
		req.HDDStandby = &v
	}
	if !plan.AdvPowerMgmt.IsNull() && !plan.AdvPowerMgmt.IsUnknown() {
		v := plan.AdvPowerMgmt.ValueString()
		req.AdvPowerMgmt = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		req.Description = &v
	}
	return req
}

// setDiskModel populates the model from what TrueNAS returned. The
// identifier stays as configured: it is the key, and the server echoes it.
func setDiskModel(model *DiskResourceModel, disk *truenas.Disk) {
	model.ID = model.Identifier
	model.HDDStandby = types.StringValue(disk.HDDStandby)
	model.AdvPowerMgmt = types.StringValue(disk.AdvPowerMgmt)
	model.Description = types.StringValue(disk.Description)
	model.Name = types.StringValue(disk.Name)
	model.Serial = types.StringValue(disk.Serial)
	model.Model = types.StringValue(disk.Model)
	model.Size = types.Int64Value(disk.Size)
	model.Type = types.StringValue(disk.Type)
	model.Pool = types.StringPointerValue(disk.Pool)
}

// apply writes the settings the plan set, if any, and stores what TrueNAS
// holds afterwards. disk.update answers without the pool join, so the disk
// is always read again.
func (r *DiskResource) apply(ctx context.Context, plan *DiskResourceModel) error {
	id := plan.Identifier.ValueString()

	if req := r.updateRequest(plan); *req != (truenas.DiskUpdateRequest{}) {
		if err := r.client.UpdateDisk(ctx, id, req); err != nil {
			return err
		}
	}

	disk, err := r.client.GetDisk(ctx, id)
	if err != nil {
		return fmt.Errorf("reading the disk back: %w", err)
	}
	setDiskModel(plan, disk)
	return nil
}

func (r *DiskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Disk start")

	var plan DiskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	// Create adopts a disk, it cannot make one, so say plainly when there
	// is none rather than letting disk.update fail on a missing row.
	if _, err := r.client.GetDisk(ctx, plan.Identifier.ValueString()); err != nil {
		if wsclient.IsNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("identifier"), "Disk Not Found",
				fmt.Sprintf("TrueNAS has no disk with identifier %q. A disk cannot be created, only adopted: "+
					"look the identifier up with the truenas_disk data source.", plan.Identifier.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Error Reading Disk", err.Error())
		return
	}

	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Creating Disk", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create Disk success")
}

func (r *DiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Disk start")

	var state DiskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	disk, err := r.client.GetDisk(ctx, state.Identifier.ValueString())
	if err != nil {
		// The disk is gone: pulled, or expired after TrueNAS stopped
		// seeing it. Dropping it from state lets the next plan say so.
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Disk", err.Error())
		return
	}

	setDiskModel(&state, disk)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read Disk success")
}

func (r *DiskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Disk start")

	var plan DiskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Updating Disk", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update Disk success")
}

// Delete only removes the disk from state and sends nothing. A disk is
// hardware that outlives the configuration: a destroy that put the standby
// timer or power level back would change how a drive in a live pool behaves,
// and nothing here may wipe or touch the data on it.
func (r *DiskResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Disk: state only, the disk and its settings are left alone")
}

func (r *DiskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identifier"), req.ID)...)
}
