package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &ServiceResource{}
	_ resource.ResourceWithImportState = &ServiceResource{}
)

// ServiceResource manages a TrueNAS service.
type ServiceResource struct {
	client *client.Client
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Service  types.String   `tfsdk:"service"`
	Enable   types.Bool     `tfsdk:"enable"`
	State    types.String   `tfsdk:"state"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

func (r *ServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a TrueNAS service (enable/disable and start/stop).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the service.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "The service name (e.g., nfs, cifs, ssh, iscsitarget, snmp, ftp, ups).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"afp", "cifs", "dynamicdns", "ftp", "iscsitarget",
						"lldp", "nfs", "openvpn_client", "openvpn_server",
						"rsync", "s3", "smartd", "snmp", "ssh", "tftp",
						"ups", "webdav", "keepalived", "netdata",
						"openipmi", "glusterd", "ctdb", "idmap", "cron",
						"smb",
					),
				},
			},
			"enable": schema.BoolAttribute{
				Description: "Whether the service is enabled to start on boot.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"state": schema.StringAttribute{
				Description: "The current state of the service (RUNNING or STOPPED).",
				Computed:    true,
			},
		},
	}
}

func (r *ServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Service start")

	var plan ServiceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceName := plan.Service.ValueString()

	tflog.Debug(ctx, "Creating service resource", map[string]interface{}{
		"service": serviceName,
	})

	// Look up the service by name
	svc, err := r.client.GetServiceByName(ctx, serviceName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Finding Service",
			fmt.Sprintf("Could not find service %q: %s", serviceName, err),
		)
		return
	}

	// Update the enable flag
	enable := plan.Enable.ValueBool()
	err = r.client.UpdateService(ctx, svc.ID, &client.ServiceUpdateRequest{
		Enable: enable,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Service",
			fmt.Sprintf("Could not update service %q: %s", serviceName, err),
		)
		return
	}

	// Start or stop the service based on enable flag
	if enable {
		if err := r.client.StartService(ctx, serviceName); err != nil {
			resp.Diagnostics.AddError(
				"Error Starting Service",
				fmt.Sprintf("Could not start service %q: %s", serviceName, err),
			)
			return
		}
	} else {
		// Stop if it happens to be running
		_ = r.client.StopService(ctx, serviceName)
	}

	// Re-read to get current state
	svc, err = r.client.GetService(ctx, svc.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Service",
			fmt.Sprintf("Could not read service %q after update: %s", serviceName, err),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(svc.ID))
	plan.Enable = types.BoolValue(svc.Enable)
	plan.State = types.StringValue(svc.State)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Service success")
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Service start")

	var state ServiceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse service ID %q: %s", state.ID.ValueString(), err))
		return
	}

	svc, err := r.client.GetService(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Service",
			fmt.Sprintf("Could not read service %d: %s", id, err),
		)
		return
	}

	state.ID = types.StringValue(strconv.Itoa(svc.ID))
	state.Service = types.StringValue(svc.Service)
	state.Enable = types.BoolValue(svc.Enable)
	state.State = types.StringValue(svc.State)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Service success")
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Service start")

	var plan ServiceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ServiceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse service ID: %s", err))
		return
	}

	serviceName := plan.Service.ValueString()
	enable := plan.Enable.ValueBool()

	err = r.client.UpdateService(ctx, id, &client.ServiceUpdateRequest{
		Enable: enable,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Service",
			fmt.Sprintf("Could not update service %q: %s", serviceName, err),
		)
		return
	}

	// Start or stop based on enable flag
	if enable {
		if err := r.client.StartService(ctx, serviceName); err != nil {
			resp.Diagnostics.AddError(
				"Error Starting Service",
				fmt.Sprintf("Could not start service %q: %s", serviceName, err),
			)
			return
		}
	} else {
		_ = r.client.StopService(ctx, serviceName)
	}

	// Re-read to get current state
	svc, err := r.client.GetService(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Service",
			fmt.Sprintf("Could not read service after update: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(svc.ID))
	plan.Enable = types.BoolValue(svc.Enable)
	plan.State = types.StringValue(svc.State)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Service success")
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Service start")

	var state ServiceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse service ID: %s", err))
		return
	}

	serviceName := state.Service.ValueString()

	tflog.Debug(ctx, "Deleting service resource (disabling and stopping)", map[string]interface{}{
		"service": serviceName,
	})

	// Stop the service
	_ = r.client.StopService(ctx, serviceName)

	// Disable the service
	err = r.client.UpdateService(ctx, id, &client.ServiceUpdateRequest{
		Enable: false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Disabling Service",
			fmt.Sprintf("Could not disable service %q: %s", serviceName, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Service success")
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import accepts either a numeric ID or a service name (e.g., "ssh",
	// "nfs"). Non-numeric input is resolved to a numeric ID via the API
	// before delegating to the standard passthrough helper so the framework
	// sets up a properly-typed null `timeouts` block. Read runs afterward
	// to populate the remaining state.
	id := req.ID
	if _, err := strconv.Atoi(id); err != nil {
		svc, lookupErr := r.client.GetServiceByName(ctx, id)
		if lookupErr != nil {
			resp.Diagnostics.AddError(
				"Error Importing Service",
				fmt.Sprintf("Could not find service %q: %s", id, lookupErr),
			)
			return
		}
		req.ID = strconv.Itoa(svc.ID)
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
