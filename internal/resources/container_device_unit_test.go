package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func containerDeviceBody(attrs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"id": 9, "container": 3, "attributes": attrs}
}

func containerDeviceClient(t *testing.T, body map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return body, nil
		})
}

func fsObject(source, target string) types.Object {
	o, _ := types.ObjectValue(containerDeviceFilesystemAttrTypes, map[string]attr.Value{
		"source": types.StringValue(source), "target": types.StringValue(target),
	})
	return o
}

func nicObject(typ, attach, mac string, trust bool) types.Object {
	o, _ := types.ObjectValue(containerDeviceNICAttrTypes, map[string]attr.Value{
		"type": types.StringValue(typ), "nic_attach": types.StringValue(attach),
		"mac": types.StringValue(mac), "trust_guest_rx_filters": types.BoolValue(trust),
	})
	return o
}

func usbObject(vendor, product, device string) types.Object {
	o, _ := types.ObjectValue(containerDeviceUSBAttrTypes, map[string]attr.Value{
		"vendor_id": types.StringValue(vendor), "product_id": types.StringValue(product),
		"device": types.StringValue(device),
	})
	return o
}

func gpuObject(gpuType, pci string) types.Object {
	o, _ := types.ObjectValue(containerDeviceGPUAttrTypes, map[string]attr.Value{
		"gpu_type": types.StringValue(gpuType), "pci_address": types.StringValue(pci),
	})
	return o
}

// emptyDeviceModel is a model with all four blocks null, so each test can
// set exactly the one it is about.
func emptyDeviceModel() ContainerDeviceResourceModel {
	return ContainerDeviceResourceModel{
		Container:  types.Int64Value(3),
		Filesystem: types.ObjectNull(containerDeviceFilesystemAttrTypes),
		GPU:        types.ObjectNull(containerDeviceGPUAttrTypes),
		NIC:        types.ObjectNull(containerDeviceNICAttrTypes),
		USB:        types.ObjectNull(containerDeviceUSBAttrTypes),
	}
}

func TestContainerDeviceResource_CRUD(t *testing.T) {
	c := containerDeviceClient(t, containerDeviceBody(map[string]interface{}{
		"dtype": "FILESYSTEM", "source": "/mnt/tank/media", "target": "/srv/media",
	}))
	r := &ContainerDeviceResource{client: c}
	crudDrive(t, r, c, "9", map[string]tftypes.Value{
		"container": tftypes.NewValue(tftypes.Number, 3),
	})
}

// The wire union is chosen by dtype, so every block must tag itself.
func TestContainerDeviceResource_buildAttributes(t *testing.T) {
	r := &ContainerDeviceResource{}

	t.Run("filesystem", func(t *testing.T) {
		m := emptyDeviceModel()
		m.Filesystem = fsObject("/mnt/tank/media", "/srv/media")
		got, err := r.buildAttributes(&m)
		if err != nil {
			t.Fatalf("buildAttributes: %v", err)
		}
		if got["dtype"] != truenas.ContainerDeviceFilesystem || got["source"] != "/mnt/tank/media" || got["target"] != "/srv/media" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("gpu", func(t *testing.T) {
		m := emptyDeviceModel()
		m.GPU = gpuObject("NVIDIA", "0000:01:00.0")
		got, err := r.buildAttributes(&m)
		if err != nil {
			t.Fatalf("buildAttributes: %v", err)
		}
		if got["dtype"] != truenas.ContainerDeviceGPU || got["gpu_type"] != "NVIDIA" {
			t.Errorf("got %v", got)
		}
	})

	// nic_attach and mac are nullable upstream, and null is what asks
	// TrueNAS to leave the interface unattached and generate a MAC. An
	// empty string is a different thing and is rejected.
	t.Run("nic sends null, not empty strings", func(t *testing.T) {
		m := emptyDeviceModel()
		m.NIC = nicObject("VIRTIO", "", "", false)
		got, err := r.buildAttributes(&m)
		if err != nil {
			t.Fatalf("buildAttributes: %v", err)
		}
		if got["dtype"] != truenas.ContainerDeviceNIC || got["type"] != "VIRTIO" {
			t.Errorf("got %v", got)
		}
		for _, k := range []string{"nic_attach", "mac"} {
			if got[k] != nil {
				t.Errorf("%s = %#v, want nil: an empty string is rejected upstream", k, got[k])
			}
		}
	})

	t.Run("nic sends what was set", func(t *testing.T) {
		m := emptyDeviceModel()
		m.NIC = nicObject("E1000", "truenasbr0", "00:11:22:33:44:55", true)
		got, err := r.buildAttributes(&m)
		if err != nil {
			t.Fatalf("buildAttributes: %v", err)
		}
		if got["nic_attach"] != "truenasbr0" || got["mac"] != "00:11:22:33:44:55" || got["trust_guest_rx_filters"] != true {
			t.Errorf("got %v", got)
		}
	})

	t.Run("usb pair becomes one nested object", func(t *testing.T) {
		m := emptyDeviceModel()
		m.USB = usbObject("0x1d6b", "0x0002", "")
		got, err := r.buildAttributes(&m)
		if err != nil {
			t.Fatalf("buildAttributes: %v", err)
		}
		usb, ok := got["usb"].(map[string]interface{})
		if !ok || usb["vendor_id"] != "0x1d6b" || usb["product_id"] != "0x0002" {
			t.Errorf("usb = %#v", got["usb"])
		}
		if got["device"] != nil {
			t.Errorf("device = %#v, want nil", got["device"])
		}
	})

	t.Run("usb by device path sends a null pair", func(t *testing.T) {
		m := emptyDeviceModel()
		m.USB = usbObject("", "", "/dev/bus/usb/001/002")
		got, err := r.buildAttributes(&m)
		if err != nil {
			t.Fatalf("buildAttributes: %v", err)
		}
		if got["usb"] != nil {
			t.Errorf("usb = %#v, want nil", got["usb"])
		}
		if got["device"] != "/dev/bus/usb/001/002" {
			t.Errorf("device = %v", got["device"])
		}
	})

	t.Run("no block set is an error", func(t *testing.T) {
		m := emptyDeviceModel()
		if _, err := r.buildAttributes(&m); err == nil {
			t.Error("a device with no block was accepted")
		}
	})
}

// Every block the server did not report must read back absent, not as an
// empty object: an empty object is a device that claims to exist.
func TestContainerDeviceResource_mapResponseSelectsOneBlock(t *testing.T) {
	r := &ContainerDeviceResource{}

	cases := []struct {
		name  string
		attrs map[string]interface{}
		set   func(m *ContainerDeviceResourceModel) types.Object
		check func(t *testing.T, m *ContainerDeviceResourceModel)
	}{
		{
			"filesystem",
			map[string]interface{}{"dtype": "FILESYSTEM", "source": "/mnt/tank/m", "target": "/srv/m"},
			func(m *ContainerDeviceResourceModel) types.Object { return m.Filesystem },
			func(t *testing.T, m *ContainerDeviceResourceModel) {
				if got, _ := m.Filesystem.Attributes()["source"].(types.String); got.ValueString() != "/mnt/tank/m" {
					t.Errorf("source = %v", m.Filesystem)
				}
			},
		},
		{
			"gpu",
			map[string]interface{}{"dtype": "GPU", "gpu_type": "AMD", "pci_address": "0000:01:00.0"},
			func(m *ContainerDeviceResourceModel) types.Object { return m.GPU },
			func(t *testing.T, m *ContainerDeviceResourceModel) {
				if got, _ := m.GPU.Attributes()["gpu_type"].(types.String); got.ValueString() != "AMD" {
					t.Errorf("gpu_type = %v", m.GPU)
				}
			},
		},
		{
			"nic",
			map[string]interface{}{"dtype": "NIC", "type": "VIRTIO", "nic_attach": "truenasbr0",
				"mac": "00:11:22:33:44:55", "trust_guest_rx_filters": true},
			func(m *ContainerDeviceResourceModel) types.Object { return m.NIC },
			func(t *testing.T, m *ContainerDeviceResourceModel) {
				a := m.NIC.Attributes()
				if got, _ := a["mac"].(types.String); got.ValueString() != "00:11:22:33:44:55" {
					t.Errorf("mac = %v", a["mac"])
				}
				if got, _ := a["trust_guest_rx_filters"].(types.Bool); !got.ValueBool() {
					t.Errorf("trust_guest_rx_filters = %v", a["trust_guest_rx_filters"])
				}
			},
		},
		{
			"usb",
			map[string]interface{}{"dtype": "USB",
				"usb":    map[string]interface{}{"vendor_id": "0x1d6b", "product_id": "0x0002"},
				"device": "/dev/bus/usb/001/002"},
			func(m *ContainerDeviceResourceModel) types.Object { return m.USB },
			func(t *testing.T, m *ContainerDeviceResourceModel) {
				a := m.USB.Attributes()
				if got, _ := a["vendor_id"].(types.String); got.ValueString() != "0x1d6b" {
					t.Errorf("vendor_id = %v", a["vendor_id"])
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m ContainerDeviceResourceModel
			err := r.mapResponseToModel(&truenas.ContainerDevice{
				ID: 9, Container: 3, Attributes: tc.attrs,
			}, &m)
			if err != nil {
				t.Fatalf("mapResponseToModel: %v", err)
			}
			if tc.set(&m).IsNull() {
				t.Fatalf("%s block was not populated", tc.name)
			}
			tc.check(t, &m)

			var others int
			for _, o := range []types.Object{m.Filesystem, m.GPU, m.NIC, m.USB} {
				if !o.IsNull() {
					others++
				}
			}
			if others != 1 {
				t.Errorf("%d blocks populated, want exactly 1", others)
			}
			if m.ID.ValueString() != "9" || m.Container.ValueInt64() != 3 {
				t.Errorf("id/container = %v/%v", m.ID, m.Container)
			}
		})
	}
}

// A USB device with no vendor/product pair reads back as empty strings
// rather than null, so a plan never shows them as "(known after apply)".
func TestContainerDeviceResource_usbNullPairReadsAsKnown(t *testing.T) {
	r := &ContainerDeviceResource{}
	var m ContainerDeviceResourceModel
	if err := r.mapResponseToModel(&truenas.ContainerDevice{
		ID: 9, Container: 3,
		Attributes: map[string]interface{}{"dtype": "USB", "usb": nil, "device": "/dev/x"},
	}, &m); err != nil {
		t.Fatalf("mapResponseToModel: %v", err)
	}
	a := m.USB.Attributes()
	for _, k := range []string{"vendor_id", "product_id"} {
		v, _ := a[k].(types.String)
		if v.IsNull() || v.IsUnknown() || v.ValueString() != "" {
			t.Errorf("%s read back as %v, want a known empty string", k, a[k])
		}
	}
}

// An unmodelled device type must be an error. Leaving all four blocks null
// would make Terraform plan the device as needing recreation on every run,
// with nothing explaining why.
func TestContainerDeviceResource_unknownDeviceTypeIsAnError(t *testing.T) {
	r := &ContainerDeviceResource{}
	var m ContainerDeviceResourceModel
	err := r.mapResponseToModel(&truenas.ContainerDevice{
		ID: 9, Container: 3, Attributes: map[string]interface{}{"dtype": "PCI"},
	}, &m)
	if err == nil {
		t.Fatal("an unmodelled device type was accepted, leaving state with no block set")
	}
	if !strings.Contains(err.Error(), "PCI") {
		t.Errorf("diagnostic should name the unknown type, got: %v", err)
	}
}

func TestContainerDeviceResource_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &ContainerDeviceResource{}
	sch := schemaOf(t, ctx, r)

	fsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"source": tftypes.String, "target": tftypes.String,
	}}
	usbType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"vendor_id": tftypes.String, "product_id": tftypes.String, "device": tftypes.String,
	}}
	gpuType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"gpu_type": tftypes.String, "pci_address": tftypes.String,
	}}
	nicType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type": tftypes.String, "nic_attach": tftypes.String,
		"mac": tftypes.String, "trust_guest_rx_filters": tftypes.Bool,
	}}
	fs := func(source, target string) tftypes.Value {
		return tftypes.NewValue(fsType, map[string]tftypes.Value{
			"source": tftypes.NewValue(tftypes.String, source),
			"target": tftypes.NewValue(tftypes.String, target),
		})
	}
	usb := func(vendor, product, device string) tftypes.Value {
		return tftypes.NewValue(usbType, map[string]tftypes.Value{
			"vendor_id":  tftypes.NewValue(tftypes.String, vendor),
			"product_id": tftypes.NewValue(tftypes.String, product),
			"device":     tftypes.NewValue(tftypes.String, device),
		})
	}
	nic := tftypes.NewValue(nicType, map[string]tftypes.Value{
		"type":                   tftypes.NewValue(tftypes.String, "E1000"),
		"nic_attach":             tftypes.NewValue(tftypes.String, ""),
		"mac":                    tftypes.NewValue(tftypes.String, ""),
		"trust_guest_rx_filters": tftypes.NewValue(tftypes.Bool, false),
	})

	// A real config leaves unset blocks NULL. stateFromValues fills every
	// attribute it is not given with an object skeleton instead, which
	// would look like all four blocks being set, so the nulls are explicit
	// here.
	config := func(set map[string]tftypes.Value) tfsdk.Config {
		vals := map[string]tftypes.Value{
			"filesystem": tftypes.NewValue(fsType, nil),
			"gpu":        tftypes.NewValue(gpuType, nil),
			"nic":        tftypes.NewValue(nicType, nil),
			"usb":        tftypes.NewValue(usbType, nil),
		}
		for k, v := range set {
			vals[k] = v
		}
		return tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, vals).Raw}
	}

	cases := []struct {
		name    string
		vals    map[string]tftypes.Value
		wantErr string
	}{
		{"one block is valid", map[string]tftypes.Value{"filesystem": fs("/mnt/tank/m", "/srv/m")}, ""},
		{"no block at all", map[string]tftypes.Value{}, "must be set"},
		{"two blocks", map[string]tftypes.Value{"filesystem": fs("/mnt/tank/m", "/srv/m"), "nic": nic}, "may be set"},
		{"source outside /mnt", map[string]tftypes.Value{"filesystem": fs("/etc/passwd", "/srv/m")}, "under /mnt/"},
		{"brace in source", map[string]tftypes.Value{"filesystem": fs("/mnt/tank/{x}", "/srv/m")}, "braces"},
		{"brace in target", map[string]tftypes.Value{"filesystem": fs("/mnt/tank/m", "/srv/{x}")}, "braces"},
		{"usb pair complete", map[string]tftypes.Value{"usb": usb("0x1d6b", "0x0002", "")}, ""},
		{"usb by device path", map[string]tftypes.Value{"usb": usb("", "", "/dev/x")}, ""},
		{"usb half a pair", map[string]tftypes.Value{"usb": usb("0x1d6b", "", "")}, "both or neither"},
		{"usb with nothing", map[string]tftypes.Value{"usb": usb("", "", "")}, "nothing behind it"},
		{"usb id not hex", map[string]tftypes.Value{"usb": usb("1d6b", "0002", "")}, "start with 0x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config(tc.vals)}, resp)
			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("valid config rejected: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatal("invalid config accepted")
			}
			var joined string
			for _, d := range resp.Diagnostics.Errors() {
				joined += d.Detail()
			}
			if !strings.Contains(joined, tc.wantErr) {
				t.Errorf("diagnostic %q does not mention %q", joined, tc.wantErr)
			}
		})
	}
}

func failingDeviceClient(t *testing.T) *wsclient.Client {
	t.Helper()
	return newWSTestClient(context.Background(), t,
		func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[EINVAL] nope"}
		})
}

func TestContainerDeviceResource_CRUDErrorsSurface(t *testing.T) {
	ctx := context.Background()
	r := &ContainerDeviceResource{client: failingDeviceClient(t)}
	sch := schemaOf(t, ctx, r)
	fsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"source": tftypes.String, "target": tftypes.String,
	}}
	vals := map[string]tftypes.Value{
		"id": str("9"), "container": tftypes.NewValue(tftypes.Number, 3),
		"filesystem": tftypes.NewValue(fsType, map[string]tftypes.Value{
			"source": tftypes.NewValue(tftypes.String, "/mnt/tank/m"),
			"target": tftypes.NewValue(tftypes.String, "/srv/m"),
		}),
	}
	st := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)

	cResp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create reported success against a server that rejected it")
	}
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read reported success against a server that returned an error")
	}
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update reported success against a server that rejected it")
	}
	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete reported success against a server that rejected it")
	}
}

// A device deleted out of band drops from state so the next plan recreates
// it, instead of failing every run. Deleting one already gone is success.
func TestContainerDeviceResource_notFoundHandling(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "[ENOENT] does not exist"}
	})
	r := &ContainerDeviceResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("9"), "container": tftypes.NewValue(tftypes.Number, 3),
	})

	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("a deleted device errored instead of dropping from state: %v", rResp.Diagnostics)
	}
	if !rResp.State.Raw.IsNull() {
		t.Error("a device deleted out of band stayed in state")
	}

	dResp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Errorf("deleting an already-absent device errored: %v", dResp.Diagnostics)
	}
}

// Detaching a device must never destroy the storage behind it.
func TestContainerDeviceResource_deleteNeverDestroysStorage(t *testing.T) {
	ctx := context.Background()
	var opts map[string]interface{}
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		if m == "container.device.delete" && len(p) > 1 {
			opts, _ = p[1].(map[string]interface{})
		}
		return true, nil
	})
	r := &ContainerDeviceResource{client: c}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("9"), "container": tftypes.NewValue(tftypes.Number, 3),
	})
	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", resp.Diagnostics)
	}
	if opts == nil {
		t.Fatal("container.device.delete was called without options")
	}
	for _, k := range []string{"raw_file", "zvol"} {
		if opts[k] != false {
			t.Errorf("%s = %v, want false: it destroys the storage behind the device", k, opts[k])
		}
	}
	if opts["force"] != true {
		t.Errorf("force = %v, want true: without it a device in use cannot be detached", opts["force"])
	}
}

// A non-numeric ID must be rejected everywhere it is parsed, rather than
// silently becoming 0 and operating on the wrong device.
func TestContainerDeviceResource_nonNumericIDRejected(t *testing.T) {
	ctx := context.Background()
	r := &ContainerDeviceResource{client: containerDeviceClient(t, containerDeviceBody(
		map[string]interface{}{"dtype": "FILESYSTEM", "source": "/mnt/t", "target": "/srv/t"}))}
	sch := schemaOf(t, ctx, r)
	bad := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("nope")})
	badPlan := planFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("nope")})

	rResp := &resource.ReadResponse{State: bad}
	r.Read(ctx, resource.ReadRequest{State: bad}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted a non-numeric ID")
	}
	uResp := &resource.UpdateResponse{State: bad}
	r.Update(ctx, resource.UpdateRequest{State: bad, Plan: badPlan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted a non-numeric ID")
	}
	dResp := &resource.DeleteResponse{State: bad}
	r.Delete(ctx, resource.DeleteRequest{State: bad}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete accepted a non-numeric ID")
	}
	iResp := &resource.ImportStateResponse{State: bad}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "nope"}, iResp)
	if !iResp.Diagnostics.HasError() {
		t.Error("ImportState accepted a non-numeric ID")
	}
	okResp := &resource.ImportStateResponse{State: bad}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "9"}, okResp)
	if okResp.Diagnostics.HasError() {
		t.Errorf("a numeric import ID was rejected: %v", okResp.Diagnostics)
	}
}

// A plan or config that fails to decode must stop the handler, not be
// applied half-read.
func TestContainerDeviceResource_undecodableInputStops(t *testing.T) {
	ctx := context.Background()
	r := &ContainerDeviceResource{client: failingDeviceClient(t)}
	sch := schemaOf(t, ctx, r)
	bogus := tftypes.NewValue(tftypes.String, "not-an-object")

	vResp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sch.Schema, Raw: bogus}}, vResp)
	if !vResp.Diagnostics.HasError() {
		t.Error("ValidateConfig accepted a config it could not decode")
	}
	cResp := &resource.CreateResponse{}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: bogus}}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create accepted a plan it could not decode")
	}
	rResp := &resource.ReadResponse{}
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: sch.Schema, Raw: bogus}}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted a state it could not decode")
	}
	uResp := &resource.UpdateResponse{}
	r.Update(ctx, resource.UpdateRequest{
		State: tfsdk.State{Schema: sch.Schema, Raw: bogus},
		Plan:  tfsdk.Plan{Schema: sch.Schema, Raw: bogus},
	}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted a plan it could not decode")
	}
	dResp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: sch.Schema, Raw: bogus}}, dResp)
	if !dResp.Diagnostics.HasError() {
		t.Error("Delete accepted a state it could not decode")
	}
	mResp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: sch.Schema, Raw: bogus}}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan: tfsdk.Plan{Schema: sch.Schema, Raw: bogus},
	}, mResp)
}

func TestDeviceObject(t *testing.T) {
	obj, err := deviceObject("FILESYSTEM", 9, containerDeviceFilesystemAttrTypes, map[string]attr.Value{
		"source": types.StringValue("/mnt/t"), "target": types.StringValue("/srv/t"),
	})
	if err != nil || obj.IsNull() {
		t.Fatalf("a well-formed block failed: %v", err)
	}
	// A value whose type does not match the attribute must be an error, not
	// a half-built object written into state.
	bad, err := deviceObject("FILESYSTEM", 9, containerDeviceFilesystemAttrTypes, map[string]attr.Value{
		"source": types.BoolValue(true), "target": types.StringValue("/srv/t"),
	})
	if err == nil {
		t.Error("a type-mismatched block was accepted")
	}
	if !bad.IsNull() {
		t.Error("a failed block returned a non-null object")
	}
}

// The server can report an attribute of a shape this provider does not
// expect. That must read as an empty string, not panic or drop the field.
func TestContainerDeviceResource_nonStringAttributeReadsAsEmpty(t *testing.T) {
	r := &ContainerDeviceResource{}
	var m ContainerDeviceResourceModel
	if err := r.mapResponseToModel(&truenas.ContainerDevice{
		ID: 9, Container: 3,
		Attributes: map[string]interface{}{"dtype": "FILESYSTEM", "source": 42, "target": nil},
	}, &m); err != nil {
		t.Fatalf("mapResponseToModel: %v", err)
	}
	a := m.Filesystem.Attributes()
	for _, k := range []string{"source", "target"} {
		v, _ := a[k].(types.String)
		if v.IsNull() || v.ValueString() != "" {
			t.Errorf("%s read back as %v, want a known empty string", k, a[k])
		}
	}
}

// Create and Update must stop when the body cannot be built, rather than
// sending a device with no type.
func TestContainerDeviceResource_buildFailureStopsWrite(t *testing.T) {
	ctx := context.Background()
	var wrote bool
	c := newWSTestClient(ctx, t, func(ctx context.Context, m string, p []interface{}) (interface{}, *wsclient.RPCError) {
		if m == "container.device.create" || m == "container.device.update" {
			wrote = true
		}
		return containerDeviceBody(map[string]interface{}{"dtype": "FILESYSTEM"}), nil
	})
	r := &ContainerDeviceResource{client: c}
	sch := schemaOf(t, ctx, r)

	fsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"source": tftypes.String, "target": tftypes.String,
	}}
	usbType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"vendor_id": tftypes.String, "product_id": tftypes.String, "device": tftypes.String,
	}}
	gpuType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"gpu_type": tftypes.String, "pci_address": tftypes.String,
	}}
	nicType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type": tftypes.String, "nic_attach": tftypes.String,
		"mac": tftypes.String, "trust_guest_rx_filters": tftypes.Bool,
	}}
	// No device block set at all: ValidateConfig would normally catch this,
	// but the handlers must not depend on that having run.
	vals := map[string]tftypes.Value{
		"id": str("9"), "container": tftypes.NewValue(tftypes.Number, 3),
		"filesystem": tftypes.NewValue(fsType, nil),
		"gpu":        tftypes.NewValue(gpuType, nil),
		"nic":        tftypes.NewValue(nicType, nil),
		"usb":        tftypes.NewValue(usbType, nil),
	}
	st := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)

	cResp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create accepted a device with no block set")
	}
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted a device with no block set")
	}
	if wrote {
		t.Error("a write went out despite the body failing to build")
	}
}

// A server that reports a device type this provider does not model must
// fail the read, not leave state with no block set.
func TestContainerDeviceResource_unknownTypeFromServerFailsCRUD(t *testing.T) {
	ctx := context.Background()
	c := containerDeviceClient(t, containerDeviceBody(map[string]interface{}{"dtype": "PCI"}))
	r := &ContainerDeviceResource{client: c}
	sch := schemaOf(t, ctx, r)

	fsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"source": tftypes.String, "target": tftypes.String,
	}}
	vals := map[string]tftypes.Value{
		"id": str("9"), "container": tftypes.NewValue(tftypes.Number, 3),
		"filesystem": tftypes.NewValue(fsType, map[string]tftypes.Value{
			"source": tftypes.NewValue(tftypes.String, "/mnt/tank/m"),
			"target": tftypes.NewValue(tftypes.String, "/srv/m"),
		}),
	}
	st := stateFromValues(t, ctx, sch, vals)
	plan := planFromValues(t, ctx, sch, vals)

	cResp := &resource.CreateResponse{State: st}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if !cResp.Diagnostics.HasError() {
		t.Error("Create accepted an unmodelled device type from the server")
	}
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Error("Read accepted an unmodelled device type from the server")
	}
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if !uResp.Diagnostics.HasError() {
		t.Error("Update accepted an unmodelled device type from the server")
	}
}

// Fields that are unknown at validate time cannot be checked, and must be
// skipped rather than compared against an empty string.
func TestContainerDeviceResource_ValidateConfigSkipsUnknownFields(t *testing.T) {
	ctx := context.Background()
	r := &ContainerDeviceResource{}
	sch := schemaOf(t, ctx, r)

	fsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"source": tftypes.String, "target": tftypes.String,
	}}
	usbType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"vendor_id": tftypes.String, "product_id": tftypes.String, "device": tftypes.String,
	}}
	gpuType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"gpu_type": tftypes.String, "pci_address": tftypes.String,
	}}
	nicType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type": tftypes.String, "nic_attach": tftypes.String,
		"mac": tftypes.String, "trust_guest_rx_filters": tftypes.Bool,
	}}

	t.Run("filesystem with an unknown source", func(t *testing.T) {
		cfg := tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, map[string]tftypes.Value{
			"filesystem": tftypes.NewValue(fsType, map[string]tftypes.Value{
				"source": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				"target": tftypes.NewValue(tftypes.String, "/srv/m"),
			}),
			"gpu": tftypes.NewValue(gpuType, nil),
			"nic": tftypes.NewValue(nicType, nil),
			"usb": tftypes.NewValue(usbType, nil),
		}).Raw}
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("an unknown source was judged instead of skipped: %v", resp.Diagnostics)
		}
	})

	t.Run("usb with an unknown vendor and a device path", func(t *testing.T) {
		cfg := tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, map[string]tftypes.Value{
			"filesystem": tftypes.NewValue(fsType, nil),
			"gpu":        tftypes.NewValue(gpuType, nil),
			"nic":        tftypes.NewValue(nicType, nil),
			"usb": tftypes.NewValue(usbType, map[string]tftypes.Value{
				"vendor_id":  tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				"product_id": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				"device":     tftypes.NewValue(tftypes.String, "/dev/x"),
			}),
		}).Raw}
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("an unknown vendor/product pair was judged instead of skipped: %v", resp.Diagnostics)
		}
	})
}
