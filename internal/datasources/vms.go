package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &VMsDataSource{}

// VMsDataSource provides a listing of all TrueNAS SCALE VMs.
type VMsDataSource struct {
	client *client.Client
}

// VMsDataSourceModel is the top-level data source model.
type VMsDataSourceModel struct {
	VMs types.List `tfsdk:"vms"`
}

func NewVMsDataSource() datasource.DataSource {
	return &VMsDataSource{}
}

func (d *VMsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vms"
}

// vmListAttrTypes returns the per-element object types for the list.
func vmListAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":     types.Int64Type,
		"name":   types.StringType,
		"memory": types.Int64Type,
		"vcpus":  types.Int64Type,
		"cores":  types.Int64Type,
		"state":  types.StringType,
	}
}

func (d *VMsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all virtual machines on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"vms": schema.ListNestedAttribute{
				Description: "List of virtual machines.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "VM ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "VM name.",
							Computed:    true,
						},
						"memory": schema.Int64Attribute{
							Description: "Memory in bytes.",
							Computed:    true,
						},
						"vcpus": schema.Int64Attribute{
							Description: "Virtual CPU count.",
							Computed:    true,
						},
						"cores": schema.Int64Attribute{
							Description: "Cores per socket.",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "Current runtime state.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *VMsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *VMsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	vms, err := d.client.ListVMs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing VMs",
			fmt.Sprintf("Could not list VMs: %s", err),
		)
		return
	}

	objType := types.ObjectType{AttrTypes: vmListAttrTypes()}
	elems := make([]attr.Value, 0, len(vms))
	for _, vm := range vms {
		state := ""
		if vm.Status != nil {
			state = vm.Status.State
		}
		obj, objDiags := types.ObjectValue(vmListAttrTypes(), map[string]attr.Value{
			"id":     types.Int64Value(int64(vm.ID)),
			"name":   types.StringValue(vm.Name),
			"memory": types.Int64Value(vm.Memory),
			"vcpus":  types.Int64Value(int64(vm.Vcpus)),
			"cores":  types.Int64Value(int64(vm.Cores)),
			"state":  types.StringValue(state),
		})
		resp.Diagnostics.Append(objDiags...)
		elems = append(elems, obj)
	}

	list, lDiags := types.ListValue(objType, elems)
	resp.Diagnostics.Append(lDiags...)

	model := VMsDataSourceModel{VMs: list}
	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}
