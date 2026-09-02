package resources

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                   = &ContainerDeviceResource{}
	_ resource.ResourceWithImportState    = &ContainerDeviceResource{}
	_ resource.ResourceWithModifyPlan     = &ContainerDeviceResource{}
	_ resource.ResourceWithValidateConfig = &ContainerDeviceResource{}
)

// ContainerDeviceResource manages one device attached to a container.
//
// Upstream models the device as a single `attributes` union discriminated
// on dtype. That is expressed here as four mutually exclusive blocks
// instead of a free-form map: the four member shapes share no fields, so a
// map would accept any key and only fail server-side, and Terraform could
// not type-check or document a single one of them.
type ContainerDeviceResource struct {
	client *wsclient.Client
}

// ContainerDeviceResourceModel describes the resource data model.
type ContainerDeviceResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Container  types.Int64    `tfsdk:"container"`
	Filesystem types.Object   `tfsdk:"filesystem"`
	GPU        types.Object   `tfsdk:"gpu"`
	NIC        types.Object   `tfsdk:"nic"`
	USB        types.Object   `tfsdk:"usb"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

// Attribute types for the four device blocks, declared once so the schema,
// the state writes and the tests cannot drift apart.
var (
	containerDeviceFilesystemAttrTypes = map[string]attr.Type{
		"target": types.StringType,
		"source": types.StringType,
	}
	containerDeviceGPUAttrTypes = map[string]attr.Type{
		"gpu_type":    types.StringType,
		"pci_address": types.StringType,
	}
	containerDeviceNICAttrTypes = map[string]attr.Type{
		"type":                   types.StringType,
		"nic_attach":             types.StringType,
		"mac":                    types.StringType,
		"trust_guest_rx_filters": types.BoolType,
	}
	containerDeviceUSBAttrTypes = map[string]attr.Type{
		"vendor_id":  types.StringType,
		"product_id": types.StringType,
		"device":     types.StringType,
	}
)

// containerDeviceBlocks is the set of mutually exclusive device blocks, in
// a fixed order so diagnostics read the same way every time.
var containerDeviceBlocks = []string{"filesystem", "gpu", "nic", "usb"}

// NewContainerDeviceResource returns a new ContainerDeviceResource factory.
func NewContainerDeviceResource() resource.Resource {
	return &ContainerDeviceResource{}
}

func (r *ContainerDeviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_device"
}

func (r *ContainerDeviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Every block is create-only in practice: changing which kind of device
	// this is means a different device entirely, and TrueNAS validates the
	// attributes against the member chosen by dtype.
	replaceObject := []planmodifier.Object{
		objectplanmodifier.UseStateForUnknown(),
		objectplanmodifier.RequiresReplace(),
	}

	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages a device attached to an LXC container on TrueNAS SCALE. Container devices " +
			"are introduced in TrueNAS 26.0; this resource cannot be used against 25.10 or older, which " +
			"have no container.device namespace. Exactly one of filesystem, gpu, nic or usb must be set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The container device ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"container": schema.Int64Attribute{
				Description: "ID of the container this device is attached to. Changing this forces a new device.",
				Required:    true,
				// Container IDs are positive: a 0 here reaches the API as a
				// container that cannot exist and comes back as a generic
				// middleware error naming nothing useful.
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"filesystem": schema.SingleNestedAttribute{
				Description:   "Bind-mount a host path into the container.",
				Optional:      true,
				PlanModifiers: replaceObject,
				Attributes: map[string]schema.Attribute{
					"source": schema.StringAttribute{
						Description: "Host path to bind-mount. Must live under /mnt, so it is on a pool rather than the boot device.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"target": schema.StringAttribute{
						Description: "Path inside the container to mount the source at.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
			"gpu": schema.SingleNestedAttribute{
				Description:   "Pass a host GPU through to the container.",
				Optional:      true,
				PlanModifiers: replaceObject,
				Attributes: map[string]schema.Attribute{
					"gpu_type": schema.StringAttribute{
						Description: "AMD, INTEL or NVIDIA.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("AMD", "INTEL", "NVIDIA"),
						},
					},
					"pci_address": schema.StringAttribute{
						Description: "PCI address of the GPU on the host.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
			"nic": schema.SingleNestedAttribute{
				Description:   "Attach a virtual network interface.",
				Optional:      true,
				PlanModifiers: replaceObject,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "E1000 for broad guest compatibility, VIRTIO for throughput.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("E1000"),
						Validators: []validator.String{
							stringvalidator.OneOf("E1000", "VIRTIO"),
						},
					},
					"nic_attach": schema.StringAttribute{
						Description: "Host bridge or MACVLAN parent to attach to, from container.device.nic_attach_choices. Empty leaves the interface unattached.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
					},
					"mac": schema.StringAttribute{
						Description: "MAC address. Generated by TrueNAS when unset.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"trust_guest_rx_filters": schema.BoolAttribute{
						Description: "Trust the guest's receive filters. Faster, and only safe when the guest is trusted.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			"usb": schema.SingleNestedAttribute{
				Description:   "Pass a host USB device through to the container.",
				Optional:      true,
				PlanModifiers: replaceObject,
				Attributes: map[string]schema.Attribute{
					"vendor_id": schema.StringAttribute{
						Description: "USB vendor ID in hex, for example 0x1d6b. Set together with product_id.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
					},
					"product_id": schema.StringAttribute{
						Description: "USB product ID in hex, for example 0x0002. Set together with vendor_id.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
					},
					"device": schema.StringAttribute{
						Description: "Host USB device path to pass through, as an alternative to the vendor/product pair.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
					},
				},
			},
		},
	}
}

func (r *ContainerDeviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig enforces the discriminated union: upstream picks the
// device shape from dtype, so exactly one block must be present. It also
// mirrors the field rules that would otherwise only fail server-side.
func (r *ContainerDeviceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg ContainerDeviceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	set := make([]string, 0, len(containerDeviceBlocks))
	for name, o := range map[string]types.Object{
		"filesystem": cfg.Filesystem,
		"gpu":        cfg.GPU,
		"nic":        cfg.NIC,
		"usb":        cfg.USB,
	} {
		if !o.IsNull() {
			set = append(set, name)
		}
	}
	sort.Strings(set)

	switch len(set) {
	case 1:
		// The single valid case.
	case 0:
		resp.Diagnostics.AddError("Invalid Device",
			fmt.Sprintf("exactly one of %s must be set; this device declares none, so there is nothing to attach",
				strings.Join(containerDeviceBlocks, ", ")))
		return
	default:
		resp.Diagnostics.AddError("Invalid Device",
			fmt.Sprintf("exactly one of %s may be set; this device declares %s. "+
				"A device is one kind of thing, so use a separate truenas_container_device for each.",
				strings.Join(containerDeviceBlocks, ", "), strings.Join(set, " and ")))
		return
	}

	r.validateFilesystem(cfg.Filesystem, resp)
	r.validateUSB(cfg.USB, resp)
}

// validateFilesystem mirrors the two rules the upstream model encodes as
// field patterns, so they fail naming the attribute.
func (r *ContainerDeviceResource) validateFilesystem(o types.Object, resp *resource.ValidateConfigResponse) {
	if o.IsNull() || o.IsUnknown() {
		return
	}
	attrs := o.Attributes()
	for _, field := range []string{"source", "target"} {
		v, ok := attrs[field].(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			continue
		}
		// Braces are rejected upstream: the path is interpolated into the
		// container config, where a brace would be taken as a template.
		if strings.ContainsAny(v.ValueString(), "{}") {
			resp.Diagnostics.AddAttributeError(path.Root("filesystem").AtName(field), "Invalid Device",
				fmt.Sprintf("%s must not contain braces, got %q", field, v.ValueString()))
		}
	}
	if src, ok := attrs["source"].(types.String); ok && !src.IsNull() && !src.IsUnknown() {
		if v := src.ValueString(); v != "" && !strings.HasPrefix(v, "/mnt/") {
			resp.Diagnostics.AddAttributeError(path.Root("filesystem").AtName("source"), "Invalid Device",
				fmt.Sprintf("source must be a path on a pool, under /mnt/, got %q. "+
					"Bind-mounting from the boot device is rejected by TrueNAS.", v))
		}
	}
}

// validateUSB rejects the half-specified vendor/product pair, which
// upstream models as one object that is present or absent.
func (r *ContainerDeviceResource) validateUSB(o types.Object, resp *resource.ValidateConfigResponse) {
	if o.IsNull() || o.IsUnknown() {
		return
	}
	attrs := o.Attributes()
	get := func(k string) string {
		v, ok := attrs[k].(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			return ""
		}
		return v.ValueString()
	}
	vendor, product, device := get("vendor_id"), get("product_id"), get("device")

	if (vendor == "") != (product == "") {
		resp.Diagnostics.AddAttributeError(path.Root("usb"), "Invalid Device",
			"vendor_id and product_id identify one device together; set both or neither.")
		return
	}
	if vendor == "" && device == "" {
		resp.Diagnostics.AddAttributeError(path.Root("usb"), "Invalid Device",
			"a usb device needs either the vendor_id and product_id pair or a device path; this sets neither, "+
				"which attaches a USB controller with nothing behind it.")
	}
	for _, f := range []struct{ name, value string }{{"vendor_id", vendor}, {"product_id", product}} {
		if f.value != "" && !strings.HasPrefix(f.value, "0x") {
			resp.Diagnostics.AddAttributeError(path.Root("usb").AtName(f.name), "Invalid Device",
				fmt.Sprintf("%s must be hexadecimal and start with 0x, for example 0x1d6b, got %q", f.name, f.value))
		}
	}
}

// ModifyPlan surfaces the destroy warning.
func (r *ContainerDeviceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_container_device")
}

// buildAttributes turns the one set block into the wire union, tagging it
// with the dtype the server discriminates on.
func (r *ContainerDeviceResource) buildAttributes(model *ContainerDeviceResourceModel) (map[string]interface{}, error) {
	str := func(attrs map[string]attr.Value, k string) string {
		v, ok := attrs[k].(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			return ""
		}
		return v.ValueString()
	}

	switch {
	case !model.Filesystem.IsNull() && !model.Filesystem.IsUnknown():
		a := model.Filesystem.Attributes()
		return map[string]interface{}{
			"dtype":  truenas.ContainerDeviceFilesystem,
			"source": str(a, "source"),
			"target": str(a, "target"),
		}, nil

	case !model.GPU.IsNull() && !model.GPU.IsUnknown():
		a := model.GPU.Attributes()
		return map[string]interface{}{
			"dtype":       truenas.ContainerDeviceGPU,
			"gpu_type":    str(a, "gpu_type"),
			"pci_address": str(a, "pci_address"),
		}, nil

	case !model.NIC.IsNull() && !model.NIC.IsUnknown():
		a := model.NIC.Attributes()
		out := map[string]interface{}{
			"dtype": truenas.ContainerDeviceNIC,
			"type":  str(a, "type"),
		}
		trust, _ := a["trust_guest_rx_filters"].(types.Bool)
		out["trust_guest_rx_filters"] = trust.ValueBool()
		// Both are nullable upstream, and null is what asks TrueNAS to
		// leave the interface unattached and to generate a MAC. An empty
		// string is not the same thing and is rejected.
		if v := str(a, "nic_attach"); v != "" {
			out["nic_attach"] = v
		} else {
			out["nic_attach"] = nil
		}
		if v := str(a, "mac"); v != "" {
			out["mac"] = v
		} else {
			out["mac"] = nil
		}
		return out, nil

	case !model.USB.IsNull() && !model.USB.IsUnknown():
		a := model.USB.Attributes()
		out := map[string]interface{}{"dtype": truenas.ContainerDeviceUSB}
		vendor, product := str(a, "vendor_id"), str(a, "product_id")
		if vendor != "" && product != "" {
			out["usb"] = map[string]interface{}{"vendor_id": vendor, "product_id": product}
		} else {
			out["usb"] = nil
		}
		if v := str(a, "device"); v != "" {
			out["device"] = v
		} else {
			out["device"] = nil
		}
		return out, nil
	}

	return nil, fmt.Errorf("no device block is set; exactly one of %s is required",
		strings.Join(containerDeviceBlocks, ", "))
}

// mapResponseToModel writes the server's view into the model, filling the
// block that matches the returned dtype and nulling the other three.
//
// An unknown dtype is an error rather than a silent no-op: leaving all four
// blocks null would make Terraform plan the device as needing recreation on
// every run, with nothing explaining why.
func (r *ContainerDeviceResource) mapResponseToModel(device *truenas.ContainerDevice, model *ContainerDeviceResourceModel) error {
	model.ID = types.StringValue(strconv.Itoa(device.ID))
	model.Container = types.Int64Value(int64(device.Container))

	model.Filesystem = types.ObjectNull(containerDeviceFilesystemAttrTypes)
	model.GPU = types.ObjectNull(containerDeviceGPUAttrTypes)
	model.NIC = types.ObjectNull(containerDeviceNICAttrTypes)
	model.USB = types.ObjectNull(containerDeviceUSBAttrTypes)

	a := device.Attributes
	dtype, _ := a["dtype"].(string)

	// Nullable upstream fields are written as known empty strings, so a
	// plan never shows them as "(known after apply)".
	s := func(k string) types.String {
		v, ok := a[k].(string)
		if !ok {
			return types.StringValue("")
		}
		return types.StringValue(v)
	}

	var (
		obj types.Object
		err error
	)
	switch dtype {
	case truenas.ContainerDeviceFilesystem:
		obj, err = deviceObject(dtype, device.ID, containerDeviceFilesystemAttrTypes, map[string]attr.Value{
			"source": s("source"),
			"target": s("target"),
		})
		model.Filesystem = obj

	case truenas.ContainerDeviceGPU:
		obj, err = deviceObject(dtype, device.ID, containerDeviceGPUAttrTypes, map[string]attr.Value{
			"gpu_type":    s("gpu_type"),
			"pci_address": s("pci_address"),
		})
		model.GPU = obj

	case truenas.ContainerDeviceNIC:
		trust, _ := a["trust_guest_rx_filters"].(bool)
		obj, err = deviceObject(dtype, device.ID, containerDeviceNICAttrTypes, map[string]attr.Value{
			"type":                   s("type"),
			"nic_attach":             s("nic_attach"),
			"mac":                    s("mac"),
			"trust_guest_rx_filters": types.BoolValue(trust),
		})
		model.NIC = obj

	case truenas.ContainerDeviceUSB:
		vendor, product := types.StringValue(""), types.StringValue("")
		if usb, ok := a["usb"].(map[string]interface{}); ok {
			if v, ok := usb["vendor_id"].(string); ok {
				vendor = types.StringValue(v)
			}
			if v, ok := usb["product_id"].(string); ok {
				product = types.StringValue(v)
			}
		}
		obj, err = deviceObject(dtype, device.ID, containerDeviceUSBAttrTypes, map[string]attr.Value{
			"vendor_id":  vendor,
			"product_id": product,
			"device":     s("device"),
		})
		model.USB = obj

	default:
		return fmt.Errorf("container device %d has device type %q, which this provider version does not model. "+
			"Upgrade the provider, or manage this device outside Terraform", device.ID, dtype)
	}
	// err is non-nil only if a value's type disagrees with its attribute,
	// which is a provider bug rather than anything the server sent. It is
	// still returned rather than dropped, so such a bug surfaces as a
	// diagnostic instead of state with an empty block.
	return err
}

// deviceObject builds one device block, turning the framework's
// diagnostics into an error so the four call sites above share a single
// failure path instead of repeating the same branch.
func deviceObject(dtype string, id int, attrTypes map[string]attr.Type, values map[string]attr.Value) (types.Object, error) {
	obj, diags := types.ObjectValue(attrTypes, values)
	if diags.HasError() {
		return types.ObjectNull(attrTypes),
			fmt.Errorf("building %s device state for container device %d: %v", dtype, id, diags.Errors())
	}
	return obj, nil
}

func (r *ContainerDeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ContainerDevice start")

	var plan ContainerDeviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	attributes, err := r.buildAttributes(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Container Device", err.Error())
		return
	}

	device, err := r.client.CreateContainerDevice(ctx, &truenas.ContainerDeviceCreateRequest{
		Container:  int(plan.Container.ValueInt64()),
		Attributes: attributes,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Container Device",
			fmt.Sprintf("Could not create device on container %d: %s", plan.Container.ValueInt64(), err),
		)
		return
	}

	if err := r.mapResponseToModel(device, &plan); err != nil {
		resp.Diagnostics.AddError("Error Creating Container Device", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create ContainerDevice success")
}

func (r *ContainerDeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ContainerDevice start")

	var state ContainerDeviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container device ID must be numeric: %s", err))
		return
	}

	device, err := r.client.GetContainerDevice(ctx, id)
	if err != nil {
		// A device deleted out of band drops from state so the next plan
		// recreates it, rather than failing every run.
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Container Device",
			fmt.Sprintf("Could not read container device %d: %s", id, err),
		)
		return
	}

	if err := r.mapResponseToModel(device, &state); err != nil {
		resp.Diagnostics.AddError("Error Reading Container Device", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read ContainerDevice success")
}

func (r *ContainerDeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ContainerDevice start")

	var plan, state ContainerDeviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container device ID must be numeric: %s", err))
		return
	}

	attributes, err := r.buildAttributes(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Container Device", err.Error())
		return
	}

	// attributes is sent whole: the union member is chosen by dtype, so a
	// partial map would be validated against a shape it does not describe.
	device, err := r.client.UpdateContainerDevice(ctx, id, &truenas.ContainerDeviceUpdateRequest{
		Attributes: attributes,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Container Device",
			fmt.Sprintf("Could not update container device %d: %s", id, err),
		)
		return
	}

	if err := r.mapResponseToModel(device, &plan); err != nil {
		resp.Diagnostics.AddError("Error Updating Container Device", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update ContainerDevice success")
}

func (r *ContainerDeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ContainerDevice start")

	var state ContainerDeviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container device ID must be numeric: %s", err))
		return
	}

	// Force detaches a device that is in use, so removal does not fail on a
	// running container. RawFile and Zvol stay false: they destroy the
	// storage behind the device, which has its own lifecycle and is not
	// something detaching a device was asked to touch.
	err = r.client.DeleteContainerDevice(ctx, id, &truenas.ContainerDeviceDeleteOptions{
		Force: true, RawFile: false, Zvol: false,
	})
	if err != nil {
		// Already gone is success: the desired end state is reached.
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "Container device already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Container Device",
			fmt.Sprintf("Could not delete container device %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ContainerDevice success")
}

func (r *ContainerDeviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container device ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
