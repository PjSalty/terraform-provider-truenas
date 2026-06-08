package resources

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// ipRegex matches IPv4 or simple IPv6 addresses. Intentionally permissive —
// server-side validation is authoritative; we only reject obvious garbage.
var ipRegex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$|^[0-9a-fA-F:]+$`)

var (
	_ resource.Resource                = &DNSNameserverResource{}
	_ resource.ResourceWithImportState = &DNSNameserverResource{}
)

// DNSNameserverResource manages DNS nameserver configuration on TrueNAS.
type DNSNameserverResource struct {
	client *client.Client
}

// DNSNameserverResourceModel describes the resource data model.
type DNSNameserverResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Nameserver1 types.String   `tfsdk:"nameserver1"`
	Nameserver2 types.String   `tfsdk:"nameserver2"`
	Nameserver3 types.String   `tfsdk:"nameserver3"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func NewDNSNameserverResource() resource.Resource {
	return &DNSNameserverResource{}
}

func (r *DNSNameserverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_nameserver"
}

func (r *DNSNameserverResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages DNS nameserver configuration on TrueNAS SCALE. " +
		"This is a singleton resource (only one instance should exist).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (always 'network_config').",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nameserver1": schema.StringAttribute{
				Description: "Primary DNS nameserver IP address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(ipRegex, "must be a valid IPv4 or IPv6 address"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nameserver2": schema.StringAttribute{
				Description: "Secondary DNS nameserver IP address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(ipRegex, "must be a valid IPv4 or IPv6 address"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nameserver3": schema.StringAttribute{
				Description: "Tertiary DNS nameserver IP address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(ipRegex, "must be a valid IPv4 or IPv6 address"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DNSNameserverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DNSNameserverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create DNSNameserver start")

	var plan DNSNameserverResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	tflog.Debug(ctx, "Setting DNS nameservers")

	config, err := r.client.UpdateNetworkConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Setting DNS Nameservers",
			fmt.Sprintf("Could not update DNS nameservers: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create DNSNameserver success")
}

func (r *DNSNameserverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read DNSNameserver start")

	var state DNSNameserverResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetNetworkConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading DNS Nameservers",
			fmt.Sprintf("Could not read network configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read DNSNameserver success")
}

func (r *DNSNameserverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update DNSNameserver start")

	var plan DNSNameserverResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateNetworkConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating DNS Nameservers",
			fmt.Sprintf("Could not update DNS nameservers: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update DNSNameserver success")
}

func (r *DNSNameserverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete DNSNameserver start")

	// For a singleton resource, "delete" clears the nameservers to empty.
	empty := ""
	updateReq := &client.NetworkConfigUpdateRequest{
		Nameserver1: &empty,
		Nameserver2: &empty,
		Nameserver3: &empty,
	}

	tflog.Debug(ctx, "Clearing DNS nameservers")

	_, err := r.client.UpdateNetworkConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Clearing DNS Nameservers",
			fmt.Sprintf("Could not clear DNS nameservers: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete DNSNameserver success")
}

func (r *DNSNameserverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DNSNameserverResource) buildUpdateRequest(model *DNSNameserverResourceModel) *client.NetworkConfigUpdateRequest {
	req := &client.NetworkConfigUpdateRequest{}

	if !model.Nameserver1.IsNull() {
		ns := model.Nameserver1.ValueString()
		req.Nameserver1 = &ns
	}
	if !model.Nameserver2.IsNull() {
		ns := model.Nameserver2.ValueString()
		req.Nameserver2 = &ns
	}
	if !model.Nameserver3.IsNull() {
		ns := model.Nameserver3.ValueString()
		req.Nameserver3 = &ns
	}

	return req
}

func (r *DNSNameserverResource) mapResponseToModel(config *client.NetworkConfig, model *DNSNameserverResourceModel) {
	model.ID = types.StringValue("network_config")
	model.Nameserver1 = types.StringValue(config.Nameserver1)
	model.Nameserver2 = types.StringValue(config.Nameserver2)
	model.Nameserver3 = types.StringValue(config.Nameserver3)
}
