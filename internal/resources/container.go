package resources

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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
	_ resource.Resource                   = &ContainerResource{}
	_ resource.ResourceWithImportState    = &ContainerResource{}
	_ resource.ResourceWithModifyPlan     = &ContainerResource{}
	_ resource.ResourceWithValidateConfig = &ContainerResource{}
)

// Idmap discriminator values, from the upstream discriminated union.
const (
	containerIdmapDefault  = "DEFAULT"
	containerIdmapIsolated = "ISOLATED"
)

// ContainerResource manages an LXC container, new in TrueNAS 26.0. The
// container namespace does not exist on 25.10 or older, so this resource
// is unusable there; the client turns that into a diagnostic naming the
// required version rather than a bare method-not-found.
type ContainerResource struct {
	client *wsclient.Client
}

// ContainerResourceModel describes the resource data model.
//
// Devices are deliberately absent: they are excluded_field() on the
// upstream create model and live in their own container.device namespace,
// so they belong to a separate resource rather than being a read-only
// mirror here that no configuration can act on.
type ContainerResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	UUID               types.String   `tfsdk:"uuid"`
	Description        types.String   `tfsdk:"description"`
	Cpuset             types.String   `tfsdk:"cpuset"`
	Autostart          types.Bool     `tfsdk:"autostart"`
	Time               types.String   `tfsdk:"time"`
	ShutdownTimeout    types.Int64    `tfsdk:"shutdown_timeout"`
	Init               types.String   `tfsdk:"init"`
	InitDir            types.String   `tfsdk:"initdir"`
	InitEnv            types.Map      `tfsdk:"initenv"`
	InitUser           types.String   `tfsdk:"inituser"`
	InitGroup          types.String   `tfsdk:"initgroup"`
	Idmap              types.Object   `tfsdk:"idmap"`
	CapabilitiesPolicy types.String   `tfsdk:"capabilities_policy"`
	CapabilitiesState  types.Map      `tfsdk:"capabilities_state"`
	Pool               types.String   `tfsdk:"pool"`
	Image              types.Object   `tfsdk:"image"`
	Dataset            types.String   `tfsdk:"dataset"`
	DefaultNetwork     types.String   `tfsdk:"default_network"`
	Status             types.Object   `tfsdk:"status"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

// Attribute types for the three nested objects, declared once so the
// schema, the state writes and the tests cannot drift apart.
var (
	containerIdmapAttrTypes = map[string]attr.Type{
		"type":  types.StringType,
		"slice": types.Int64Type,
	}
	containerImageAttrTypes = map[string]attr.Type{
		"name":    types.StringType,
		"version": types.StringType,
	}
	containerStatusAttrTypes = map[string]attr.Type{
		"state":        types.StringType,
		"pid":          types.Int64Type,
		"domain_state": types.StringType,
	}
)

// NewContainerResource returns a new ContainerResource factory.
func NewContainerResource() resource.Resource {
	return &ContainerResource{}
}

func (r *ContainerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *ContainerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages an LXC container on TrueNAS SCALE. Containers are introduced in " +
			"TrueNAS 26.0; this resource cannot be used against 25.10 or older, which have no " +
			"container namespace. Devices are managed by truenas_container_device.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The container ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Container name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			// Create-only upstream: uuid is excluded from the update model,
			// so changing it can only be expressed as a replacement.
			"uuid": schema.StringAttribute{
				Description: "Container UUID used by libvirt. Generated by TrueNAS when not set. Changing this forces a new container.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Container description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"cpuset": schema.StringAttribute{
				Description: "Physical CPU numbers the container's processes and virtual CPUs may be pinned to, for example \"0-3\". Empty means no pinning.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"autostart": schema.BoolAttribute{
				Description: "Start the container automatically on boot. Defaults to true, matching the server.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"time": schema.StringAttribute{
				Description: "Whether container time is local time or UTC.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LOCAL"),
				Validators: []validator.String{
					stringvalidator.OneOf("LOCAL", "UTC"),
				},
			},
			"shutdown_timeout": schema.Int64Attribute{
				Description: "Seconds to wait for the container to shut down before killing it.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(90),
			},
			"init": schema.StringAttribute{
				Description: "Command line of the container's init process.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/sbin/init"),
			},
			"initdir": schema.StringAttribute{
				Description: "Working directory of the init process. Empty means the image default.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"initenv": schema.MapAttribute{
				Description: "Environment variables for the init process.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"inituser": schema.StringAttribute{
				Description: "Username the init process runs as. Empty means the image default.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"initgroup": schema.StringAttribute{
				Description: "Group the init process runs as. Empty means the image default.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			// Create-only upstream: idmap is excluded from the update model.
			"idmap": schema.SingleNestedAttribute{
				Description: "User-namespace ID mapping. Omit for the TrueNAS default. Changing this forces a new container. " +
					"Leaving the block out is NOT the same as an unmapped container: there is no way to express " +
					"an unmapped container here, by design, because that maps container root to host root.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "DEFAULT applies the standard TrueNAS mapping. ISOLATED offsets it per container so no two containers share host UIDs.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(containerIdmapDefault, containerIdmapIsolated),
						},
					},
					"slice": schema.Int64Attribute{
						Description: "ISOLATED only. The offset multiplier, below 1000. Omit to have TrueNAS pick an unused one.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"capabilities_policy": schema.StringAttribute{
				Description: "DEFAULT drops sys_module, sys_time, mknod, audit_control and mac_admin. ALLOW keeps every capability. DENY drops all of them.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DEFAULT"),
				Validators: []validator.String{
					stringvalidator.OneOf("DEFAULT", "ALLOW", "DENY"),
				},
			},
			"capabilities_state": schema.MapAttribute{
				Description: "Per-capability overrides on top of capabilities_policy, keyed by capability name.",
				ElementType: types.BoolType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			// Create-only upstream: pool is excluded from the update model.
			"pool": schema.StringAttribute{
				Description: "Pool hosting the container's root filesystem. Empty uses the pool from truenas_lxc_config. Changing this forces a new container.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Create-only upstream: image is excluded from the update model.
			"image": schema.SingleNestedAttribute{
				Description: "Image the container is created from. Changing this forces a new container.",
				Required:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "Image name, as listed by container.image.query_registry.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"version": schema.StringAttribute{
						Description: "Image version, as listed by container.image.query_registry.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
			// Derived server-side, reported and never sent.
			"dataset": schema.StringAttribute{
				Description: "Dataset used as the container root filesystem. Derived by TrueNAS, read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_network": schema.StringAttribute{
				Description: "Bridge the container uses when no NIC device is attached. Empty once NIC devices are attached, because the configuration is then on the devices. Derived by TrueNAS, read-only.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.SingleNestedAttribute{
				Description: "Runtime state. Derived by TrueNAS, read-only. This resource does not start or stop containers; use autostart, or the TrueNAS UI/API for an ad-hoc run.",
				Computed:    true,
				// Without this the framework replans every Computed
				// attribute as unknown on any update, and an update that
				// touches nothing still reports a diff forever. Refresh
				// still rewrites these from the server, so pinning the
				// planned value does not hide real drift.
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"state": schema.StringAttribute{
						Description: "RUNNING, STOPPED or SUSPENDED.",
						Computed:    true,
					},
					"pid": schema.Int64Attribute{
						Description: "Host PID of the container's init process while running, 0 otherwise. Informational only.",
						Computed:    true,
					},
					"domain_state": schema.StringAttribute{
						Description: "Domain state as reported by libvirt.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *ContainerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig mirrors the cross-field rules the upstream discriminated
// union encodes, so they fail at plan time naming the attribute rather
// than at apply time as a middleware validation error.
func (r *ContainerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg ContainerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !cfg.ShutdownTimeout.IsNull() && !cfg.ShutdownTimeout.IsUnknown() {
		if v := cfg.ShutdownTimeout.ValueInt64(); v < 5 || v > 300 {
			resp.Diagnostics.AddAttributeError(path.Root("shutdown_timeout"), "Invalid Shutdown Timeout",
				fmt.Sprintf("shutdown_timeout must be between 5 and 300 seconds, got %d", v))
		}
	}

	if cfg.Idmap.IsNull() || cfg.Idmap.IsUnknown() {
		return
	}
	attrs := cfg.Idmap.Attributes()
	idmapType, _ := attrs["type"].(types.String)
	slice, _ := attrs["slice"].(types.Int64)
	if idmapType.IsNull() || idmapType.IsUnknown() {
		return
	}
	// slice is meaningful only for ISOLATED. Accepting it on DEFAULT would
	// let a config express an offset the server silently ignores.
	if idmapType.ValueString() == containerIdmapDefault && !slice.IsNull() && !slice.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("idmap").AtName("slice"), "Invalid Idmap",
			fmt.Sprintf("slice only applies to an %s idmap; %s takes no offset. Remove slice, or set type = %q.",
				containerIdmapIsolated, containerIdmapDefault, containerIdmapIsolated))
	}
	if idmapType.ValueString() == containerIdmapIsolated && !slice.IsNull() && !slice.IsUnknown() {
		if v := slice.ValueInt64(); v < 1 || v >= 1000 {
			resp.Diagnostics.AddAttributeError(path.Root("idmap").AtName("slice"), "Invalid Idmap",
				fmt.Sprintf("slice must be between 1 and 999, got %d. Omit it to have TrueNAS pick an unused one.", v))
		}
	}
}

// ModifyPlan surfaces the destroy warning, and keeps an imported
// container from being replaced on its first apply.
//
// image and pool exist only on the upstream CREATE model: ContainerEntry
// carries neither, so no read can ever report them. On a normal refresh
// that is harmless, Read leaves the prior state values alone. After
// `terraform import` there are no prior values at all, so state holds null
// while the configuration names an image and a pool, and a RequiresReplace
// attribute turns that into destroy-and-recreate of a container the
// operator just adopted.
//
// Since the server cannot tell us which image or pool a container came
// from, the only honest handling is to accept what the configuration says:
// drop the replace for exactly those attributes, and only while state has
// no value to compare against. A later change to either still replaces
// normally, because by then state holds what the last apply set.
func (r *ContainerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_container")
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		// Destroy or create: there is no adopted-state case to handle.
		return
	}

	var state ContainerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	adopted := make([]path.Path, 0, 2)
	if containerImageUnset(state.Image) {
		adopted = append(adopted, path.Root("image"))
	}
	if state.Pool.IsNull() || state.Pool.IsUnknown() || state.Pool.ValueString() == "" {
		adopted = append(adopted, path.Root("pool"))
	}
	if len(adopted) == 0 {
		return
	}

	kept := make([]path.Path, 0, len(resp.RequiresReplace))
	for _, p := range resp.RequiresReplace {
		var carved bool
		for _, a := range adopted {
			if p.Equal(a) {
				carved = true
				break
			}
		}
		if carved {
			continue
		}
		kept = append(kept, p)
	}
	resp.RequiresReplace = kept
	tflog.Debug(ctx, "container has create-only attributes missing from state (imported); accepting the configured values without replacing",
		map[string]interface{}{"attributes": fmt.Sprint(adopted)})
}

// buildCreateRequest turns a plan into a container.create body. Only
// attributes the plan actually set are sent, so an omitted attribute takes
// the server's default rather than a zero value chosen here.
func (r *ContainerResource) buildCreateRequest(ctx context.Context, plan *ContainerResourceModel) (*truenas.ContainerCreateRequest, error) {
	req := &truenas.ContainerCreateRequest{Name: plan.Name.ValueString()}

	img := plan.Image.Attributes()
	name, _ := img["name"].(types.String)
	version, _ := img["version"].(types.String)
	req.Image = truenas.ContainerImageRef{Name: name.ValueString(), Version: version.ValueString()}

	setStr := func(v types.String, dst **string) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		s := v.ValueString()
		*dst = &s
	}
	// uuid and pool are "unset means let TrueNAS decide", so an empty
	// string must be omitted rather than sent as "".
	setOptionalStr := func(v types.String, dst **string) {
		if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
			return
		}
		s := v.ValueString()
		*dst = &s
	}

	setOptionalStr(plan.UUID, &req.UUID)
	setOptionalStr(plan.Pool, &req.Pool)
	setStr(plan.Description, &req.Description)
	setStr(plan.Cpuset, &req.Cpuset)
	setStr(plan.Time, &req.Time)
	setStr(plan.Init, &req.Init)
	setStr(plan.InitDir, &req.InitDir)
	setStr(plan.InitUser, &req.InitUser)
	setStr(plan.InitGroup, &req.InitGroup)
	setStr(plan.CapabilitiesPolicy, &req.CapabilitiesPolicy)

	if !plan.Autostart.IsNull() && !plan.Autostart.IsUnknown() {
		v := plan.Autostart.ValueBool()
		req.Autostart = &v
	}
	if !plan.ShutdownTimeout.IsNull() && !plan.ShutdownTimeout.IsUnknown() {
		v := int(plan.ShutdownTimeout.ValueInt64())
		req.ShutdownTimeout = &v
	}

	env, err := containerStringMap(ctx, plan.InitEnv)
	if err != nil {
		return nil, fmt.Errorf("reading initenv: %w", err)
	}
	req.InitEnv = env

	caps, err := containerBoolMap(ctx, plan.CapabilitiesState)
	if err != nil {
		return nil, fmt.Errorf("reading capabilities_state: %w", err)
	}
	req.CapabilitiesState = caps

	req.Idmap = containerIdmapFromObject(plan.Idmap)
	return req, nil
}

// buildUpdateRequest turns a plan into a container.update body. pool,
// image and idmap are absent because upstream excludes them from the
// update model; the schema marks them RequiresReplace for the same reason.
func (r *ContainerResource) buildUpdateRequest(ctx context.Context, plan *ContainerResourceModel) (*truenas.ContainerUpdateRequest, error) {
	req := &truenas.ContainerUpdateRequest{}

	setStr := func(v types.String, dst **string) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		s := v.ValueString()
		*dst = &s
	}
	setStr(plan.Name, &req.Name)
	setStr(plan.Description, &req.Description)
	setStr(plan.Cpuset, &req.Cpuset)
	setStr(plan.Time, &req.Time)
	setStr(plan.Init, &req.Init)
	setStr(plan.InitDir, &req.InitDir)
	setStr(plan.InitUser, &req.InitUser)
	setStr(plan.InitGroup, &req.InitGroup)
	setStr(plan.CapabilitiesPolicy, &req.CapabilitiesPolicy)

	if !plan.Autostart.IsNull() && !plan.Autostart.IsUnknown() {
		v := plan.Autostart.ValueBool()
		req.Autostart = &v
	}
	if !plan.ShutdownTimeout.IsNull() && !plan.ShutdownTimeout.IsUnknown() {
		v := int(plan.ShutdownTimeout.ValueInt64())
		req.ShutdownTimeout = &v
	}

	env, err := containerStringMap(ctx, plan.InitEnv)
	if err != nil {
		return nil, fmt.Errorf("reading initenv: %w", err)
	}
	req.InitEnv = env

	caps, err := containerBoolMap(ctx, plan.CapabilitiesState)
	if err != nil {
		return nil, fmt.Errorf("reading capabilities_state: %w", err)
	}
	req.CapabilitiesState = caps
	return req, nil
}

// containerStringMap converts a plan map into a pointer the encoder omits
// when the attribute was not set.
//
// Note for anyone porting this to a LIST attribute: ElementsAs yields a
// nil SLICE for an empty input, and a non-nil pointer to a nil slice
// marshals as JSON null, which middleware rejects as not-a-list. Maps do
// not have that problem, ElementsAs always allocates, so no nil guard is
// needed here.
//
// request field is `omitempty`, and encoding/json omits an EMPTY map as
// readily as a nil one. A plain map would therefore make `initenv = {}`
// indistinguishable from "not set", silently leaving the old environment in
// place instead of clearing it. A pointer omits only when it is nil.
//
//nolint:gocritic // The pointer is load-bearing, not a style choice: the
func containerStringMap(ctx context.Context, m types.Map) (*map[string]string, error) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]string{}
	if diags := m.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil, fmt.Errorf("%v", diags.Errors())
	}
	return &out, nil
}

// containerBoolMap is containerStringMap for capabilities_state.
//
//nolint:gocritic // Pointer for the same reason as containerStringMap.
func containerBoolMap(ctx context.Context, m types.Map) (*map[string]bool, error) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]bool{}
	if diags := m.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil, fmt.Errorf("%v", diags.Errors())
	}
	return &out, nil
}

// containerIdmapFromObject reads the idmap block into the wire union.
//
// The slice key is discriminator-sensitive. ISOLATED must always carry it,
// null meaning "pick an unused one", because upstream declares it with no
// default and pydantic then requires the key. DEFAULT must never carry it,
// because that member forbids extras. Leaving it out of ISOLATED is what made
// `idmap = { type = "ISOLATED" }`, the provider's own published example,
// fail with `container_create.idmap.ISOLATED.slice: Field required`.
func containerIdmapFromObject(o types.Object) *truenas.ContainerIdmap {
	if o.IsNull() || o.IsUnknown() {
		return nil
	}
	attrs := o.Attributes()
	t, _ := attrs["type"].(types.String)
	if t.IsNull() || t.IsUnknown() {
		return nil
	}
	out := &truenas.ContainerIdmap{Type: t.ValueString()}
	if out.Type != containerIdmapIsolated {
		return out
	}
	var slice *int
	if s, ok := attrs["slice"].(types.Int64); ok && !s.IsNull() && !s.IsUnknown() {
		v := int(s.ValueInt64())
		slice = &v
	}
	out.Slice = &slice
	return out
}

// mapResponseToModel writes the server's view into the model.
//
// Every nullable upstream field is written as a known value rather than
// null, so a plan does not show it as "(known after apply)" on every run.
func (r *ContainerResource) mapResponseToModel(ctx context.Context, c *truenas.Container, model *ContainerResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(c.ID))
	model.UUID = types.StringValue(c.UUID)
	model.Name = types.StringValue(c.Name)
	model.Description = types.StringValue(c.Description)
	model.Cpuset = containerStr(c.Cpuset)
	model.Autostart = types.BoolValue(c.Autostart)
	model.Time = types.StringValue(c.Time)
	model.ShutdownTimeout = types.Int64Value(int64(c.ShutdownTimeout))
	model.Dataset = types.StringValue(c.Dataset)
	model.Init = types.StringValue(c.Init)
	model.InitDir = containerStr(c.InitDir)
	model.InitUser = containerStr(c.InitUser)
	model.InitGroup = containerStr(c.InitGroup)
	model.CapabilitiesPolicy = types.StringValue(c.CapabilitiesPolicy)
	model.DefaultNetwork = containerStr(c.DefaultNetwork)

	env := map[string]string{}
	if c.InitEnv != nil {
		env = c.InitEnv
	}
	envVal, diags := types.MapValueFrom(ctx, types.StringType, env)
	if !diags.HasError() {
		model.InitEnv = envVal
	}

	caps := map[string]bool{}
	if c.CapabilitiesState != nil {
		caps = c.CapabilitiesState
	}
	capsVal, diags := types.MapValueFrom(ctx, types.BoolType, caps)
	if !diags.HasError() {
		model.CapabilitiesState = capsVal
	}

	// A null idmap upstream means "no user namespace at all". The schema
	// cannot express it, so it is reported as a null object rather than
	// silently rendered as DEFAULT, which would read as safer than the
	// container actually is.
	if c.Idmap == nil {
		model.Idmap = types.ObjectNull(containerIdmapAttrTypes)
	} else {
		// Slice is a double pointer: absent, present-and-null, or a number.
		// The first two both read back as a null Int64, which is right,
		// because the server assigning a slice is what fills it in.
		slice := types.Int64Null()
		if c.Idmap.Slice != nil && *c.Idmap.Slice != nil {
			slice = types.Int64Value(int64(**c.Idmap.Slice))
		}
		obj, diags := types.ObjectValue(containerIdmapAttrTypes, map[string]attr.Value{
			"type":  types.StringValue(c.Idmap.Type),
			"slice": slice,
		})
		if !diags.HasError() {
			model.Idmap = obj
		}
	}

	pid := types.Int64Value(0)
	if c.Status.PID != nil {
		pid = types.Int64Value(int64(*c.Status.PID))
	}
	statusObj, diags := types.ObjectValue(containerStatusAttrTypes, map[string]attr.Value{
		"state":        types.StringValue(c.Status.State),
		"pid":          pid,
		"domain_state": containerStr(c.Status.DomainState),
	})
	if !diags.HasError() {
		model.Status = statusObj
	}
}

// containerImageUnset reports whether state carries no usable image.
//
// Straight after an import the whole object is null. Defensive against the
// other shape too: an object whose name is null or empty is not an image
// anyone can compare against, and treating it as one would force a replace
// on a container the operator just adopted.
func containerImageUnset(o types.Object) bool {
	if o.IsNull() || o.IsUnknown() {
		return true
	}
	name, ok := o.Attributes()["name"].(types.String)
	return !ok || name.IsNull() || name.IsUnknown() || name.ValueString() == ""
}

// containerStr renders a nullable upstream string as a known empty string.
func containerStr(p *string) types.String {
	if p == nil {
		return types.StringValue("")
	}
	return types.StringValue(*p)
}

func (r *ContainerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Container start")

	var plan ContainerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	createReq, err := r.buildCreateRequest(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Container", err.Error())
		return
	}

	// The pool must exist before the image pull starts, or the failure
	// arrives minutes later from inside the job with less context.
	if createReq.Pool != nil {
		if err := r.validatePool(ctx, *createReq.Pool); err != nil {
			resp.Diagnostics.AddError("Error Creating Container", err.Error())
			return
		}
	}

	container, err := r.client.CreateContainer(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Container",
			fmt.Sprintf("Could not create container %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, container, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create Container success")
}

// validatePool rejects a pool the container service will not accept,
// listing the ones it will.
func (r *ContainerResource) validatePool(ctx context.Context, pool string) error {
	choices, err := r.client.GetContainerPoolChoices(ctx)
	if err != nil {
		return fmt.Errorf("fetching pool choices for validation: %w", err)
	}
	if _, ok := choices[pool]; ok {
		return nil
	}
	names := make([]string, 0, len(choices))
	for name := range choices {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("pool %q cannot host containers, and this system reports no usable pools "+
			"(a locked or boot pool is never offered)", pool)
	}
	return fmt.Errorf("pool %q cannot host containers; available pools: %s", pool, strings.Join(names, ", "))
}

func (r *ContainerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Container start")

	var state ContainerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container ID must be numeric: %s", err))
		return
	}

	container, err := r.client.GetContainer(ctx, id)
	if err != nil {
		// A container deleted out of band drops from state so the next
		// plan recreates it, rather than failing every run.
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Container",
			fmt.Sprintf("Could not read container %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, container, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read Container success")
}

func (r *ContainerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Container start")

	var plan, state ContainerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container ID must be numeric: %s", err))
		return
	}

	updateReq, err := r.buildUpdateRequest(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Container", err.Error())
		return
	}

	container, err := r.client.UpdateContainer(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Container",
			fmt.Sprintf("Could not update container %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, container, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update Container success")
}

func (r *ContainerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Container start")

	var state ContainerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container ID must be numeric: %s", err))
		return
	}

	// Force stops a running container so the destroy does not fail on it.
	// Recursive stays false: it would destroy the container's child
	// datasets and snapshots, any clones of those snapshots anywhere in
	// the pool, and any holds on them, none of it recoverable and none of
	// it something Terraform was asked to manage.
	err = r.client.DeleteContainer(ctx, id, &truenas.ContainerDeleteOptions{Force: true, Recursive: false})
	if err != nil {
		// Already gone is success: the desired end state is reached.
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "Container already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Container",
			fmt.Sprintf("Could not delete container %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Container success")
}

func (r *ContainerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Container ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
