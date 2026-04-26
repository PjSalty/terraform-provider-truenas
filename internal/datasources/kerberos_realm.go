package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &KerberosRealmDataSource{}

// KerberosRealmDataSource provides information about a Kerberos realm.
type KerberosRealmDataSource struct {
	client *client.Client
}

// KerberosRealmDataSourceModel describes the data source model.
type KerberosRealmDataSourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Realm         types.String `tfsdk:"realm"`
	PrimaryKDC    types.String `tfsdk:"primary_kdc"`
	KDC           types.List   `tfsdk:"kdc"`
	AdminServer   types.List   `tfsdk:"admin_server"`
	KPasswdServer types.List   `tfsdk:"kpasswd_server"`
}

func NewKerberosRealmDataSource() datasource.DataSource {
	return &KerberosRealmDataSource{}
}

func (d *KerberosRealmDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kerberos_realm"
}

func (d *KerberosRealmDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a Kerberos realm on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric realm ID to look up.",
				Required:    true,
			},
			"realm": schema.StringAttribute{
				Description: "The realm name.",
				Computed:    true,
			},
			"primary_kdc": schema.StringAttribute{
				Description: "Primary KDC server.",
				Computed:    true,
			},
			"kdc": schema.ListAttribute{
				Description: "KDC servers.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"admin_server": schema.ListAttribute{
				Description: "Admin servers.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"kpasswd_server": schema.ListAttribute{
				Description: "kpasswd servers.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *KerberosRealmDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KerberosRealmDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KerberosRealmDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r, err := d.client.GetKerberosRealm(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Kerberos Realm",
			fmt.Sprintf("Could not read kerberos realm with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(r.ID))
	config.Realm = types.StringValue(r.Realm)
	if r.PrimaryKDC != nil {
		config.PrimaryKDC = types.StringValue(*r.PrimaryKDC)
	} else {
		config.PrimaryKDC = types.StringNull()
	}

	kdc, kDiags := types.ListValueFrom(ctx, types.StringType, r.KDC)
	resp.Diagnostics.Append(kDiags...)
	config.KDC = kdc

	admin, aDiags := types.ListValueFrom(ctx, types.StringType, r.AdminServer)
	resp.Diagnostics.Append(aDiags...)
	config.AdminServer = admin

	kpw, kpDiags := types.ListValueFrom(ctx, types.StringType, r.KPasswdServer)
	resp.Diagnostics.Append(kpDiags...)
	config.KPasswdServer = kpw

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
