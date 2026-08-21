package resources

import (
	"context"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// lxcConfigSingletonID is the fixed identifier for truenas_lxc_config.
// TrueNAS has exactly one LXC configuration, so the ID is a constant.
const lxcConfigSingletonID = "lxc_config"

// lxcBridgeAuto is the sentinel lxc.bridge_choices returns for "let
// TrueNAS create and manage the bridge". TrueNAS stores it as a null,
// which refreshes back as "", so a config spelling the bridge "[AUTO]"
// would never converge. The provider cannot silently rewrite it either:
// for an Optional+Computed attribute the framework requires the planned
// value to equal a non-null config value, so normalising it in ModifyPlan
// is rejected as "provider produced invalid plan". The only correct
// handling is to reject it at validate time and name the spelling that
// works.
const lxcBridgeAuto = "[AUTO]"

var (
	_ resource.Resource                   = &LXCConfigResource{}
	_ resource.ResourceWithImportState    = &LXCConfigResource{}
	_ resource.ResourceWithValidateConfig = &LXCConfigResource{}
)

// LXCConfigResource manages the system-wide LXC container configuration,
// new in TrueNAS 26.0. The lxc namespace does not exist on 25.10 or older.
type LXCConfigResource struct {
	client *wsclient.Client
}

// LXCConfigResourceModel describes the resource data model.
type LXCConfigResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	PreferredPool types.String   `tfsdk:"preferred_pool"`
	Bridge        types.String   `tfsdk:"bridge"`
	V4Network     types.String   `tfsdk:"v4_network"`
	V6Network     types.String   `tfsdk:"v6_network"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

// NewLXCConfigResource returns a new LXCConfigResource factory.
func NewLXCConfigResource() resource.Resource {
	return &LXCConfigResource{}
}

func (r *LXCConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lxc_config"
}

func (r *LXCConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
		Description: "Manages the system-wide LXC container configuration on TrueNAS SCALE: the default " +
			"pool for container and image datasets, and the bridge and networks used for container " +
			"networking. This is a singleton, TrueNAS has exactly one LXC configuration. Requires " +
			"TrueNAS 26.0 or newer.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Fixed singleton identifier. Always \"lxc_config\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"preferred_pool": schema.StringAttribute{
				Description: "Default pool used for container and image datasets. Empty means unset, " +
					"which is a real state: TrueNAS then has no default and container creation must " +
					"name a pool.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bridge": schema.StringAttribute{
				Description: "Network bridge interface for container networking. Empty means TrueNAS " +
					"manages and creates one automatically. Validated against lxc.bridge_choices at " +
					"apply time, so an interface that does not exist is rejected with the valid " +
					"choices listed rather than failing server-side.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"v4_network": schema.StringAttribute{
				Description: "IPv4 network CIDR for the container bridge network.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"v6_network": schema.StringAttribute{
				Description: "IPv6 network CIDR for the container bridge network.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *LXCConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig mirrors the checks lxc.update runs server-side, so a bad
// CIDR fails at plan time with a message naming the attribute rather than
// at apply time as a generic middleware validation error.
func (r *LXCConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg LXCConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateNet := func(attr string, v types.String, want4 bool) {
		if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
			return
		}
		prefix, err := netip.ParsePrefix(v.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Invalid Network",
				fmt.Sprintf("%s must be a CIDR network such as 172.200.0.0/24: %s", attr, err))
			return
		}
		if want4 != prefix.Addr().Is4() {
			family := "IPv4"
			if !want4 {
				family = "IPv6"
			}
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Invalid Network",
				fmt.Sprintf("%s must be an %s network, got %q", attr, family, v.ValueString()))
			return
		}
		// TrueNAS rejects anything smaller than 4 addresses: the bridge
		// needs a network address, a gateway and at least one guest.
		bits := prefix.Addr().BitLen() - prefix.Bits()
		if bits < 2 {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Invalid Network",
				fmt.Sprintf("%s must have at least 4 addresses, %q has %d",
					attr, v.ValueString(), 1<<bits))
		}
	}

	validateNet("v4_network", cfg.V4Network, true)
	validateNet("v6_network", cfg.V6Network, false)

	if !cfg.Bridge.IsNull() && !cfg.Bridge.IsUnknown() && cfg.Bridge.ValueString() == lxcBridgeAuto {
		resp.Diagnostics.AddAttributeError(path.Root("bridge"), "Invalid Bridge",
			fmt.Sprintf("%q is how the API advertises the automatic bridge in lxc.bridge_choices, "+
				"but TrueNAS stores it as a null that reads back as an empty string, so a "+
				"configuration using it can never converge. Set bridge = \"\" (or omit it) to "+
				"have TrueNAS manage the bridge.", lxcBridgeAuto))
	}
}

// applyConfig is the shared write path for Create and Update.
//
// Only attributes the plan actually set are sent. lxc.update is a
// ForUpdateMetaclass model, so an omitted key leaves the stored value
// alone; sending an empty string instead would clear a setting the config
// never mentioned.
func (r *LXCConfigResource) applyConfig(ctx context.Context, plan *LXCConfigResourceModel) error {
	req := &truenas.LXCConfigUpdateRequest{}

	if !plan.PreferredPool.IsNull() && !plan.PreferredPool.IsUnknown() {
		v := plan.PreferredPool.ValueString()
		req.PreferredPool = &v
	}
	if !plan.V4Network.IsNull() && !plan.V4Network.IsUnknown() {
		v := plan.V4Network.ValueString()
		req.V4Network = &v
	}
	if !plan.V6Network.IsNull() && !plan.V6Network.IsUnknown() {
		v := plan.V6Network.ValueString()
		req.V6Network = &v
	}

	if !plan.Bridge.IsNull() && !plan.Bridge.IsUnknown() {
		want := plan.Bridge.ValueString()
		// An empty bridge means "let TrueNAS manage one", which is valid
		// and names no interface, so it is not checked against the list.
		if want != "" {
			choices, err := r.client.GetLXCBridgeChoices(ctx)
			if err != nil {
				return fmt.Errorf("fetching bridge choices for validation: %w", err)
			}
			if _, ok := choices[want]; !ok {
				return fmt.Errorf("bridge %q does not exist on this system; available bridges: %s",
					want, lxcBridgeNames(choices))
			}
		}
		req.Bridge = &want
	}

	if _, err := r.client.SetLXCConfig(ctx, req); err != nil {
		return fmt.Errorf("applying LXC config: %w", err)
	}
	return nil
}

// lxcBridgeNames renders the available bridges for a diagnostic.
func lxcBridgeNames(choices map[string]string) string {
	names := make([]string, 0, len(choices))
	for name := range choices {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none; leave bridge empty to have TrueNAS manage one)"
	}
	return strings.Join(names, ", ")
}

// refreshState populates the model from live TrueNAS state.
func (r *LXCConfigResource) refreshState(ctx context.Context, model *LXCConfigResourceModel) error {
	cfg, err := r.client.GetLXCConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading LXC config: %w", err)
	}
	// preferred_pool and bridge are nullable upstream. They are written as
	// known empty strings rather than null so the attributes stay known and
	// a plan does not show them as "(known after apply)" every run.
	model.PreferredPool = types.StringValue("")
	if cfg.PreferredPool != nil {
		model.PreferredPool = types.StringValue(*cfg.PreferredPool)
	}
	model.Bridge = types.StringValue("")
	if cfg.Bridge != nil {
		model.Bridge = types.StringValue(*cfg.Bridge)
	}
	model.V4Network = types.StringValue(cfg.V4Network)
	model.V6Network = types.StringValue(cfg.V6Network)
	model.ID = types.StringValue(lxcConfigSingletonID)
	return nil
}

func (r *LXCConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create LXCConfig start")

	var plan LXCConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.applyConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Creating LXC Config", err.Error())
		return
	}
	if err := r.refreshState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Refreshing LXC Config State", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create LXCConfig success")
}

func (r *LXCConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read LXCConfig start")

	var state LXCConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.refreshState(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error Reading LXC Config", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read LXCConfig success")
}

func (r *LXCConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update LXCConfig start")

	var plan LXCConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.applyConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Updating LXC Config", err.Error())
		return
	}
	if err := r.refreshState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error Refreshing LXC Config State", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update LXCConfig success")
}

// Delete is a no-op. The LXC configuration is a singleton that always
// exists; removing the resource stops Terraform managing it rather than
// resetting the system's container networking.
func (r *LXCConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete LXCConfig: no-op, singleton config is not removable")
}

func (r *LXCConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != lxcConfigSingletonID {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("truenas_lxc_config is a singleton; the import ID must be %q, got %q",
				lxcConfigSingletonID, req.ID),
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
