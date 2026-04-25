package resources

// Network config singleton — manages global network settings including
// hostname, domain, DNS servers, default gateways, and HTTP proxy. Backed
// by the TrueNAS SCALE /network/configuration endpoint.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	tnvalidators "github.com/PjSalty/terraform-provider-truenas/internal/validators"
)

var (
	_ resource.Resource                = &NetworkConfigResource{}
	_ resource.ResourceWithImportState = &NetworkConfigResource{}
)

// NetworkConfigResource manages the full network configuration on TrueNAS.
type NetworkConfigResource struct {
	client *client.Client
}

// NetworkConfigResourceModel describes the resource data model.
type NetworkConfigResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Hostname    types.String   `tfsdk:"hostname"`
	Domain      types.String   `tfsdk:"domain"`
	IPv4Gateway types.String   `tfsdk:"ipv4gateway"`
	IPv6Gateway types.String   `tfsdk:"ipv6gateway"`
	Nameserver1 types.String   `tfsdk:"nameserver1"`
	Nameserver2 types.String   `tfsdk:"nameserver2"`
	Nameserver3 types.String   `tfsdk:"nameserver3"`
	HTTPProxy   types.String   `tfsdk:"httpproxy"`
	Hosts       types.List     `tfsdk:"hosts"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func NewNetworkConfigResource() resource.Resource {
	return &NetworkConfigResource{}
}

func (r *NetworkConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_config"
}

func (r *NetworkConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the global network configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "The system hostname.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 63),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "The system domain.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
			},
			"ipv4gateway": schema.StringAttribute{
				Description: "IPv4 default gateway.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					tnvalidators.IPOrCIDR(),
				},
			},
			"ipv6gateway": schema.StringAttribute{
				Description: "IPv6 default gateway.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					tnvalidators.IPOrCIDR(),
				},
			},
			"nameserver1": schema.StringAttribute{
				Description: "Primary DNS nameserver.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					tnvalidators.IPOrCIDR(),
				},
			},
			"nameserver2": schema.StringAttribute{
				Description: "Secondary DNS nameserver.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					tnvalidators.IPOrCIDR(),
				},
			},
			"nameserver3": schema.StringAttribute{
				Description: "Tertiary DNS nameserver.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					tnvalidators.IPOrCIDR(),
				},
			},
			"httpproxy": schema.StringAttribute{
				Description: "HTTP proxy URL.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"hosts": schema.ListAttribute{
				Description: "Additional hosts entries.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *NetworkConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NetworkConfig start")

	var plan NetworkConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating network config resource (updating singleton)")

	var d diag.Diagnostics
	updateReq := r.buildUpdateRequest(ctx, &plan, &d)
	resp.Diagnostics.Append(d...)

	config, err := r.client.UpdateFullNetworkConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Network Config",
			fmt.Sprintf("Could not update network configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NetworkConfig success")
}

func (r *NetworkConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NetworkConfig start")

	var state NetworkConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetFullNetworkConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Config",
			fmt.Sprintf("Could not read network configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NetworkConfig success")
}

func (r *NetworkConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NetworkConfig start")

	var plan NetworkConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var d diag.Diagnostics
	updateReq := r.buildUpdateRequest(ctx, &plan, &d)
	resp.Diagnostics.Append(d...)

	config, err := r.client.UpdateFullNetworkConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Network Config",
			fmt.Sprintf("Could not update network configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NetworkConfig success")
}

func (r *NetworkConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NetworkConfig start")

	tflog.Debug(ctx, "Deleting network config resource (resetting to defaults)")

	hostname := "truenas"
	domain := "local"
	empty := ""

	_, err := r.client.UpdateFullNetworkConfig(ctx, &client.FullNetworkConfigUpdateRequest{
		Hostname:    &hostname,
		Domain:      &domain,
		IPv4Gateway: &empty,
		IPv6Gateway: &empty,
		Nameserver1: &empty,
		Nameserver2: &empty,
		Nameserver3: &empty,
		HTTPProxy:   &empty,
		Hosts:       []string{},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting Network Config",
			fmt.Sprintf("Could not reset network configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NetworkConfig success")
}

func (r *NetworkConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NetworkConfigResource) buildUpdateRequest(ctx context.Context, plan *NetworkConfigResourceModel, d *diag.Diagnostics) *client.FullNetworkConfigUpdateRequest {
	updateReq := &client.FullNetworkConfigUpdateRequest{}

	if !plan.Hostname.IsNull() && !plan.Hostname.IsUnknown() {
		v := plan.Hostname.ValueString()
		updateReq.Hostname = &v
	}
	if !plan.Domain.IsNull() && !plan.Domain.IsUnknown() {
		v := plan.Domain.ValueString()
		updateReq.Domain = &v
	}
	if !plan.IPv4Gateway.IsNull() && !plan.IPv4Gateway.IsUnknown() {
		v := plan.IPv4Gateway.ValueString()
		updateReq.IPv4Gateway = &v
	}
	if !plan.IPv6Gateway.IsNull() && !plan.IPv6Gateway.IsUnknown() {
		v := plan.IPv6Gateway.ValueString()
		updateReq.IPv6Gateway = &v
	}
	if !plan.Nameserver1.IsNull() && !plan.Nameserver1.IsUnknown() {
		v := plan.Nameserver1.ValueString()
		updateReq.Nameserver1 = &v
	}
	if !plan.Nameserver2.IsNull() && !plan.Nameserver2.IsUnknown() {
		v := plan.Nameserver2.ValueString()
		updateReq.Nameserver2 = &v
	}
	if !plan.Nameserver3.IsNull() && !plan.Nameserver3.IsUnknown() {
		v := plan.Nameserver3.ValueString()
		updateReq.Nameserver3 = &v
	}
	if !plan.HTTPProxy.IsNull() && !plan.HTTPProxy.IsUnknown() {
		v := plan.HTTPProxy.ValueString()
		updateReq.HTTPProxy = &v
	}
	if !plan.Hosts.IsNull() && !plan.Hosts.IsUnknown() {
		var hosts []string
		diags := plan.Hosts.ElementsAs(ctx, &hosts, false)
		d.Append(diags...)
		updateReq.Hosts = hosts
	}

	return updateReq
}

func (r *NetworkConfigResource) mapResponseToModel(ctx context.Context, config *client.FullNetworkConfig, model *NetworkConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.Hostname = types.StringValue(config.Hostname)
	model.Domain = types.StringValue(config.Domain)
	model.IPv4Gateway = types.StringValue(config.IPv4Gateway)
	model.IPv6Gateway = types.StringValue(config.IPv6Gateway)
	model.Nameserver1 = types.StringValue(config.Nameserver1)
	model.Nameserver2 = types.StringValue(config.Nameserver2)
	model.Nameserver3 = types.StringValue(config.Nameserver3)
	model.HTTPProxy = types.StringValue(config.HTTPProxy)

	hostsValues, diags := types.ListValueFrom(ctx, types.StringType, config.Hosts)
	if !diags.HasError() {
		model.Hosts = hostsValues
	}
}
