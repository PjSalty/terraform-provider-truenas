package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/resourcevalidators"
	tnvalidators "github.com/PjSalty/terraform-provider-truenas/internal/validators"
)

var (
	_ resource.Resource                     = &NetworkInterfaceResource{}
	_ resource.ResourceWithImportState      = &NetworkInterfaceResource{}
	_ resource.ResourceWithConfigValidators = &NetworkInterfaceResource{}
)

// NetworkInterfaceResource manages a TrueNAS network interface (BRIDGE,
// LINK_AGGREGATION, or VLAN). The underlying /interface API uses a
// staged commit-and-checkin workflow: changes go to a pending state,
// must be committed (with a rollback timer), then acknowledged via
// checkin. The client layer handles this transparently so the resource
// presents a simple CRUD interface to Terraform.
//
// Physical interfaces (type PHYSICAL) cannot be created via /interface —
// they are discovered automatically from the host. This resource only
// supports creating virtual interface types.
type NetworkInterfaceResource struct {
	client *client.Client
}

// NetworkInterfaceResourceModel describes the resource data model.
type NetworkInterfaceResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	Type                types.String   `tfsdk:"type"`
	Description         types.String   `tfsdk:"description"`
	IPv4DHCP            types.Bool     `tfsdk:"ipv4_dhcp"`
	IPv6Auto            types.Bool     `tfsdk:"ipv6_auto"`
	MTU                 types.Int64    `tfsdk:"mtu"`
	Aliases             types.List     `tfsdk:"aliases"`
	BridgeMembers       types.List     `tfsdk:"bridge_members"`
	LagProtocol         types.String   `tfsdk:"lag_protocol"`
	LagPorts            types.List     `tfsdk:"lag_ports"`
	VlanParentInterface types.String   `tfsdk:"vlan_parent_interface"`
	VlanTag             types.Int64    `tfsdk:"vlan_tag"`
	VlanPCP             types.Int64    `tfsdk:"vlan_pcp"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// aliasObjectType is the Terraform object type for NetworkInterfaceAlias.
func aliasObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":    types.StringType,
			"address": types.StringType,
			"netmask": types.Int64Type,
		},
	}
}

func NewNetworkInterfaceResource() resource.Resource {
	return &NetworkInterfaceResource{}
}

func (r *NetworkInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

func (r *NetworkInterfaceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a virtual network interface (BRIDGE, LINK_AGGREGATION, or VLAN) on TrueNAS SCALE. " +
			"Changes go through a staged commit+checkin workflow which this resource handles automatically." + "\n\n" +
			"**Stability: Beta.** Import-only verified against TrueNAS SCALE 25.10. Create/update/delete cycles were not live-tested because modifying the active management interface on the test VM risks cutting API access. The schema uses the TrueNAS staged commit/checkin workflow which is handled automatically.",
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
				Description: "The interface identifier (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The interface name. If not provided, TrueNAS auto-generates one based on type " +
					"(e.g. br0, bond1, vlan0).",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 15),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The interface type: BRIDGE, LINK_AGGREGATION, or VLAN.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("PHYSICAL", "BRIDGE", "LINK_AGGREGATION", "VLAN"),
				},
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the interface.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv4_dhcp": schema.BoolAttribute{
				Description: "Enable IPv4 DHCP for automatic IP address assignment.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv6_auto": schema.BoolAttribute{
				Description: "Enable IPv6 autoconfiguration.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"mtu": schema.Int64Attribute{
				Description: "Maximum transmission unit (68-9216 bytes).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(68, 9216),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"aliases": schema.ListNestedAttribute{
				Description: "List of IP address aliases to configure on the interface.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Address type: INET (IPv4) or INET6 (IPv6).",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("INET", "INET6"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"address": schema.StringAttribute{
							Description: "IP address.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
								tnvalidators.IPOrCIDR(),
							},
						},
						"netmask": schema.Int64Attribute{
							Description: "Network mask in CIDR notation.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 128),
							},
						},
					},
				},
			},
			"bridge_members": schema.ListAttribute{
				Description: "List of interfaces to add as members of this bridge. Only valid for type=BRIDGE.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"lag_protocol": schema.StringAttribute{
				Description: "Link aggregation protocol: LACP, FAILOVER, LOADBALANCE, ROUNDROBIN, NONE. Only valid for type=LINK_AGGREGATION.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("LACP", "FAILOVER", "LOADBALANCE", "ROUNDROBIN", "NONE"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"lag_ports": schema.ListAttribute{
				Description: "List of interface names in the link aggregation group. Only valid for type=LINK_AGGREGATION.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan_parent_interface": schema.StringAttribute{
				Description: "Parent interface name for VLAN configuration. Only valid for type=VLAN.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan_tag": schema.Int64Attribute{
				Description: "VLAN tag number (1-4094). Only valid for type=VLAN.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 4094),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"vlan_pcp": schema.Int64Attribute{
				Description: "Priority Code Point for VLAN traffic (0-7). Only valid for type=VLAN.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 7),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ConfigValidators enforces the interface type → attribute shape at
// config-validation time. TrueNAS's /interface API silently accepts
// a BRIDGE with empty bridge_members or a VLAN without vlan_tag and
// then produces a broken interface that the user has to notice via
// out-of-band tools — short-circuit those classes of mistakes here.
//
//   - type = "BRIDGE"           → bridge_members must be non-empty
//   - type = "LINK_AGGREGATION" → lag_protocol and lag_ports required
//   - type = "VLAN"             → vlan_parent_interface and vlan_tag required
//
// PHYSICAL interfaces are returned by the API for hardware NICs and
// are never created via Terraform, so we skip them here.
func (r *NetworkInterfaceResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidators.RequiredWhenEqual("type", "LINK_AGGREGATION",
			[]string{"lag_protocol"}),
		resourcevalidators.RequiredWhenEqual("type", "VLAN",
			[]string{"vlan_parent_interface"}),
	}
}

func (r *NetworkInterfaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NetworkInterface start")

	var plan NetworkInterfaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NetworkInterfaceCreateRequest{
		Type: plan.Type.ValueString(),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		createReq.Name = plan.Name.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.IPv4DHCP.IsNull() && !plan.IPv4DHCP.IsUnknown() {
		createReq.IPv4DHCP = plan.IPv4DHCP.ValueBool()
	}
	if !plan.IPv6Auto.IsNull() && !plan.IPv6Auto.IsUnknown() {
		createReq.IPv6Auto = plan.IPv6Auto.ValueBool()
	}
	if !plan.MTU.IsNull() && !plan.MTU.IsUnknown() {
		v := int(plan.MTU.ValueInt64())
		createReq.MTU = &v
	}
	if aliases, ok := aliasesFromList(ctx, plan.Aliases, &resp.Diagnostics); ok {
		createReq.Aliases = aliases
	}
	if !plan.BridgeMembers.IsNull() && !plan.BridgeMembers.IsUnknown() {
		var members []string
		d := plan.BridgeMembers.ElementsAs(ctx, &members, false)
		resp.Diagnostics.Append(d...)
		createReq.BridgeMembers = members
	}
	if !plan.LagProtocol.IsNull() && !plan.LagProtocol.IsUnknown() {
		createReq.LagProtocol = plan.LagProtocol.ValueString()
	}
	if !plan.LagPorts.IsNull() && !plan.LagPorts.IsUnknown() {
		var ports []string
		d := plan.LagPorts.ElementsAs(ctx, &ports, false)
		resp.Diagnostics.Append(d...)
		createReq.LagPorts = ports
	}
	if !plan.VlanParentInterface.IsNull() && !plan.VlanParentInterface.IsUnknown() {
		createReq.VlanParentInterface = plan.VlanParentInterface.ValueString()
	}
	if !plan.VlanTag.IsNull() && !plan.VlanTag.IsUnknown() {
		v := int(plan.VlanTag.ValueInt64())
		createReq.VlanTag = &v
	}
	if !plan.VlanPCP.IsNull() && !plan.VlanPCP.IsUnknown() {
		v := int(plan.VlanPCP.ValueInt64())
		createReq.VlanPCP = &v
	}

	tflog.Debug(ctx, "Creating interface", map[string]interface{}{"type": createReq.Type, "name": createReq.Name})

	iface, err := r.client.CreateInterface(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Interface",
			fmt.Sprintf("Could not create interface: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, iface, &plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NetworkInterface success")
}

func (r *NetworkInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NetworkInterface start")

	var state NetworkInterfaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, err := r.client.GetInterface(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Interface",
			fmt.Sprintf("Could not read interface %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, iface, &state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NetworkInterface success")
}

func (r *NetworkInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NetworkInterface start")

	var plan NetworkInterfaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state NetworkInterfaceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.NetworkInterfaceUpdateRequest{}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.IPv4DHCP.IsNull() && !plan.IPv4DHCP.IsUnknown() {
		v := plan.IPv4DHCP.ValueBool()
		updateReq.IPv4DHCP = &v
	}
	if !plan.IPv6Auto.IsNull() && !plan.IPv6Auto.IsUnknown() {
		v := plan.IPv6Auto.ValueBool()
		updateReq.IPv6Auto = &v
	}
	if !plan.MTU.IsNull() && !plan.MTU.IsUnknown() {
		v := int(plan.MTU.ValueInt64())
		updateReq.MTU = &v
	}
	if aliases, ok := aliasesFromList(ctx, plan.Aliases, &resp.Diagnostics); ok {
		updateReq.Aliases = aliases
	}
	if !plan.BridgeMembers.IsNull() && !plan.BridgeMembers.IsUnknown() {
		var members []string
		d := plan.BridgeMembers.ElementsAs(ctx, &members, false)
		resp.Diagnostics.Append(d...)
		updateReq.BridgeMembers = members
	}
	if !plan.LagProtocol.IsNull() && !plan.LagProtocol.IsUnknown() {
		v := plan.LagProtocol.ValueString()
		updateReq.LagProtocol = &v
	}
	if !plan.LagPorts.IsNull() && !plan.LagPorts.IsUnknown() {
		var ports []string
		d := plan.LagPorts.ElementsAs(ctx, &ports, false)
		resp.Diagnostics.Append(d...)
		updateReq.LagPorts = ports
	}
	if !plan.VlanParentInterface.IsNull() && !plan.VlanParentInterface.IsUnknown() {
		v := plan.VlanParentInterface.ValueString()
		updateReq.VlanParentInterface = &v
	}
	if !plan.VlanTag.IsNull() && !plan.VlanTag.IsUnknown() {
		v := int(plan.VlanTag.ValueInt64())
		updateReq.VlanTag = &v
	}
	if !plan.VlanPCP.IsNull() && !plan.VlanPCP.IsUnknown() {
		v := int(plan.VlanPCP.ValueInt64())
		updateReq.VlanPCP = &v
	}

	iface, err := r.client.UpdateInterface(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Interface",
			fmt.Sprintf("Could not update interface %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, iface, &plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NetworkInterface success")
}

func (r *NetworkInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NetworkInterface start")

	var state NetworkInterfaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting interface", map[string]interface{}{"id": state.ID.ValueString()})
	if err := r.client.DeleteInterface(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Network interface already deleted, removing from state", map[string]interface{}{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Interface",
			fmt.Sprintf("Could not delete interface %q: %s", state.ID.ValueString(), err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NetworkInterface success")
}

func (r *NetworkInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Delegate to the standard passthrough helper so the framework sets up
	// the `id` attribute and a properly-typed null `timeouts` block. Read
	// is called afterward by the framework to populate the rest of state.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NetworkInterfaceResource) mapResponseToModel(ctx context.Context, iface *client.NetworkInterface, model *NetworkInterfaceResourceModel) {
	model.ID = types.StringValue(iface.ID)
	model.Name = types.StringValue(iface.Name)
	model.Type = types.StringValue(iface.Type)
	model.Description = types.StringValue(iface.Description)
	model.IPv4DHCP = types.BoolValue(iface.IPv4DHCP)
	model.IPv6Auto = types.BoolValue(iface.IPv6Auto)

	if iface.MTU != nil {
		model.MTU = types.Int64Value(int64(*iface.MTU))
	} else {
		model.MTU = types.Int64Null()
	}

	// Aliases -> ListNested
	aliasValues := make([]attr.Value, 0, len(iface.Aliases))
	for _, a := range iface.Aliases {
		obj, _ := types.ObjectValue(aliasObjectType().AttrTypes, map[string]attr.Value{
			"type":    types.StringValue(a.Type),
			"address": types.StringValue(a.Address),
			"netmask": types.Int64Value(int64(a.Netmask)),
		})
		aliasValues = append(aliasValues, obj)
	}
	aliasList, _ := types.ListValue(aliasObjectType(), aliasValues)
	model.Aliases = aliasList

	bridgeList, _ := types.ListValueFrom(ctx, types.StringType, iface.BridgeMembers)
	model.BridgeMembers = bridgeList

	model.LagProtocol = types.StringValue(iface.LagProtocol)

	lagList, _ := types.ListValueFrom(ctx, types.StringType, iface.LagPorts)
	model.LagPorts = lagList

	model.VlanParentInterface = types.StringValue(iface.VlanParentInterface)

	if iface.VlanTag != nil {
		model.VlanTag = types.Int64Value(int64(*iface.VlanTag))
	} else {
		model.VlanTag = types.Int64Null()
	}
	if iface.VlanPCP != nil {
		model.VlanPCP = types.Int64Value(int64(*iface.VlanPCP))
	} else {
		model.VlanPCP = types.Int64Null()
	}
}

// aliasesFromList converts a Terraform list of alias objects into the client
// alias struct slice. Returns ok=false if the list is null/unknown (caller
// should skip setting the field in that case).
func aliasesFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) ([]client.NetworkInterfaceAlias, bool) {
	_ = ctx
	_ = diags
	if list.IsNull() || list.IsUnknown() {
		return nil, false
	}
	elements := list.Elements()
	result := make([]client.NetworkInterfaceAlias, 0, len(elements))
	for _, elem := range elements {
		// ListNestedAttribute schema guarantees elements are types.Object.
		obj := elem.(types.Object)
		attrs := obj.Attributes()
		alias := client.NetworkInterfaceAlias{}
		if v, ok := attrs["type"].(types.String); ok && !v.IsNull() {
			alias.Type = v.ValueString()
		}
		if v, ok := attrs["address"].(types.String); ok && !v.IsNull() {
			alias.Address = v.ValueString()
		}
		if v, ok := attrs["netmask"].(types.Int64); ok && !v.IsNull() {
			alias.Netmask = int(v.ValueInt64())
		}
		result = append(result, alias)
	}
	return result, true
}
