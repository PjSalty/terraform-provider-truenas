package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &VMDeviceResource{}
	_ resource.ResourceWithImportState = &VMDeviceResource{}
)

// VMDeviceResource manages a device attached to a TrueNAS SCALE virtual machine.
type VMDeviceResource struct {
	client *client.Client
}

// VMDeviceResourceModel describes the resource data model. The `attributes` map
// holds type-specific fields (DISK, NIC, CDROM, DISPLAY, RAW, PCI, USB) as
// strings; see the TrueNAS API docs for the fields supported by each dtype.
type VMDeviceResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	VM         types.Int64    `tfsdk:"vm"`
	Dtype      types.String   `tfsdk:"dtype"`
	Order      types.Int64    `tfsdk:"order"`
	Attributes types.Map      `tfsdk:"attributes"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

// NewVMDeviceResource returns the resource implementation for truenas_vm_device.
func NewVMDeviceResource() resource.Resource {
	return &VMDeviceResource{}
}

func (r *VMDeviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_device"
}

func (r *VMDeviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a device attached to a TrueNAS SCALE virtual machine. " +
			"The dtype field selects the device type (DISK, NIC, CDROM, DISPLAY, RAW, PCI, USB) " +
			"and the attributes map carries the type-specific fields.",
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
				Description: "The numeric ID of the VM device.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm": schema.Int64Attribute{
				Description: "The ID of the VM this device is attached to.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"dtype": schema.StringAttribute{
				Description: "The device type: DISK, NIC, CDROM, DISPLAY, RAW, PCI, or USB. " +
					"Changing this forces replacement.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("DISK", "NIC", "CDROM", "DISPLAY", "RAW", "PCI", "USB"),
				},
			},
			"order": schema.Int64Attribute{
				Description: "Device order on the VM's bus. Optional.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"attributes": schema.MapAttribute{
				Description: "Type-specific device attributes as string values. " +
					"For example DISK uses path, type (AHCI/VIRTIO), iotype; NIC uses type, mac, nic_attach; " +
					"DISPLAY uses resolution, bind, port, password, web; CDROM uses path; " +
					"RAW uses path, type, size; PCI uses pptdev; USB uses controller_type, usb. " +
					"DISPLAY devices may contain a VNC/SPICE password — marked sensitive.",
				Required:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *VMDeviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// vmDeviceAttrsToAPI converts a TF map into the API attributes map, coercing
// values that look like integers or booleans to native JSON types so the
// TrueNAS schema validator accepts them.
func vmDeviceAttrsToAPI(ctx context.Context, dtype string, tfMap types.Map) map[string]interface{} {
	attrs := map[string]interface{}{"dtype": dtype}
	if tfMap.IsNull() || tfMap.IsUnknown() {
		return attrs
	}

	// ElementsAs into map[string]string cannot fail when the schema is
	// map(string) — any type mismatch would be caught earlier at Plan.Get.
	var raw map[string]string
	_ = tfMap.ElementsAs(ctx, &raw, false)

	for k, v := range raw {
		if k == "dtype" {
			continue
		}
		// Coerce simple primitives: true/false -> bool, integer -> int.
		switch v {
		case "true":
			attrs[k] = true
			continue
		case "false":
			attrs[k] = false
			continue
		}
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			attrs[k] = i
			continue
		}
		attrs[k] = v
	}
	return attrs
}

func (r *VMDeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create VMDevice start")

	var plan VMDeviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := vmDeviceAttrsToAPI(ctx, plan.Dtype.ValueString(), plan.Attributes)

	createReq := &client.VMDeviceCreateRequest{
		VM:         int(plan.VM.ValueInt64()),
		Attributes: attrs,
	}
	if !plan.Order.IsNull() && !plan.Order.IsUnknown() {
		o := int(plan.Order.ValueInt64())
		createReq.Order = &o
	}

	tflog.Debug(ctx, "Creating VM device", map[string]interface{}{
		"vm":    createReq.VM,
		"dtype": plan.Dtype.ValueString(),
	})

	dev, err := r.client.CreateVMDevice(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VM Device",
			fmt.Sprintf("Could not create VM device: %s", err))
		return
	}

	r.mapResponseToModel(ctx, dev, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create VMDevice success")
}

func (r *VMDeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read VMDevice start")

	var state VMDeviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VM device ID: %s", err))
		return
	}

	dev, err := r.client.GetVMDevice(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VM Device",
			fmt.Sprintf("Could not read VM device %d: %s", id, err))
		return
	}

	r.mapResponseToModel(ctx, dev, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read VMDevice success")
}

func (r *VMDeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update VMDevice start")

	var plan VMDeviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state VMDeviceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VM device ID: %s", err))
		return
	}

	attrs := vmDeviceAttrsToAPI(ctx, plan.Dtype.ValueString(), plan.Attributes)

	vm := int(plan.VM.ValueInt64())
	updateReq := &client.VMDeviceUpdateRequest{
		VM:         &vm,
		Attributes: attrs,
	}
	if !plan.Order.IsNull() && !plan.Order.IsUnknown() {
		o := int(plan.Order.ValueInt64())
		updateReq.Order = &o
	}

	dev, err := r.client.UpdateVMDevice(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating VM Device",
			fmt.Sprintf("Could not update VM device %d: %s", id, err))
		return
	}

	r.mapResponseToModel(ctx, dev, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update VMDevice success")
}

func (r *VMDeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete VMDevice start")

	var state VMDeviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VM device ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting VM device", map[string]interface{}{"id": id})

	if err := r.client.DeleteVMDevice(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "VM device already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting VM Device",
			fmt.Sprintf("Could not delete VM device %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete VMDevice success")
}

func (r *VMDeviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("VM device ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VMDeviceResource) mapResponseToModel(ctx context.Context, dev *client.VMDevice, model *VMDeviceResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(dev.ID))
	model.VM = types.Int64Value(int64(dev.VM))
	if dev.Order != nil {
		model.Order = types.Int64Value(int64(*dev.Order))
	} else {
		model.Order = types.Int64Null()
	}

	// Capture the user-supplied attribute keys BEFORE overwriting the model, so
	// we can filter the server response down to just those keys. TrueNAS fills
	// in device-specific defaults (e.g. DISPLAY grows `web_port`, DISK grows
	// `logical_sectorsize`, etc.) that the user never wrote — without filtering
	// we'd hit "inconsistent result after apply" every Create.
	priorKeys := map[string]bool{}
	if !model.Attributes.IsNull() && !model.Attributes.IsUnknown() {
		var priorMap map[string]string
		if diags := model.Attributes.ElementsAs(ctx, &priorMap, false); !diags.HasError() {
			for k := range priorMap {
				priorKeys[k] = true
			}
		}
	}

	// Extract dtype from the attributes map and convert the rest to a string map
	// for TF state.
	attrMap := make(map[string]string)
	var dtype string
	for k, v := range dev.Attributes {
		if k == "dtype" {
			if s, ok := v.(string); ok {
				dtype = s
			}
			continue
		}
		if v == nil {
			continue
		}
		// Filter to user-supplied keys only (when we have a prior reference).
		// On Import (priorKeys empty) we keep everything.
		if len(priorKeys) > 0 && !priorKeys[k] {
			continue
		}
		switch val := v.(type) {
		case string:
			attrMap[k] = val
		case bool:
			attrMap[k] = strconv.FormatBool(val)
		case float64:
			// JSON numbers decode as float64; preserve integer semantics where possible.
			if val == float64(int64(val)) {
				attrMap[k] = strconv.FormatInt(int64(val), 10)
			} else {
				attrMap[k] = fmt.Sprintf("%v", val)
			}
		default:
			attrMap[k] = fmt.Sprintf("%v", val)
		}
	}

	model.Dtype = types.StringValue(dtype)
	mapVal, _ := types.MapValueFrom(ctx, types.StringType, attrMap)
	model.Attributes = mapVal
}
