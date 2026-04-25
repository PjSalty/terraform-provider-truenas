package datasources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &CertificateDataSource{}

// CertificateDataSource provides information about a TrueNAS certificate.
type CertificateDataSource struct {
	client *client.Client
}

// CertificateDataSourceModel describes the data source model.
type CertificateDataSourceModel struct {
	ID              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Issuer          types.String `tfsdk:"issuer"`
	ValidFrom       types.String `tfsdk:"valid_from"`
	ValidUntil      types.String `tfsdk:"valid_until"`
	SAN             types.String `tfsdk:"san"`
	KeyType         types.String `tfsdk:"key_type"`
	KeyLength       types.Int64  `tfsdk:"key_length"`
	DigestAlgorithm types.String `tfsdk:"digest_algorithm"`
	Expired         types.Bool   `tfsdk:"expired"`
	DN              types.String `tfsdk:"dn"`
}

func NewCertificateDataSource() datasource.DataSource {
	return &CertificateDataSource{}
}

func (d *CertificateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (d *CertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a certificate on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The certificate ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The certificate name to look up.",
				Required:    true,
			},
			"issuer": schema.StringAttribute{
				Description: "The certificate issuer (common name).",
				Computed:    true,
			},
			"valid_from": schema.StringAttribute{
				Description: "Certificate validity start date.",
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: "Certificate validity end date.",
				Computed:    true,
			},
			"san": schema.StringAttribute{
				Description: "Subject Alternative Names (comma-separated).",
				Computed:    true,
			},
			"key_type": schema.StringAttribute{
				Description: "The key type (RSA, EC).",
				Computed:    true,
			},
			"key_length": schema.Int64Attribute{
				Description: "The key length in bits.",
				Computed:    true,
			},
			"digest_algorithm": schema.StringAttribute{
				Description: "The digest algorithm (e.g., SHA256).",
				Computed:    true,
			},
			"expired": schema.BoolAttribute{
				Description: "Whether the certificate has expired.",
				Computed:    true,
			},
			"dn": schema.StringAttribute{
				Description: "The certificate distinguished name.",
				Computed:    true,
			},
		},
	}
}

func (d *CertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CertificateDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()

	cert, err := d.client.GetCertificateByName(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Certificate",
			fmt.Sprintf("Could not find certificate %q: %s", name, err),
		)
		return
	}

	config.ID = types.Int64Value(int64(cert.ID))
	config.Name = types.StringValue(cert.Name)
	config.ValidFrom = types.StringValue(cert.From)
	config.ValidUntil = types.StringValue(cert.Until)
	config.KeyType = types.StringValue(cert.KeyType)
	config.KeyLength = types.Int64Value(int64(cert.KeyLength))
	config.DigestAlgorithm = types.StringValue(cert.DigestAlgorithm)
	config.Expired = types.BoolValue(cert.Expired)
	config.DN = types.StringValue(cert.DN)

	config.Issuer = types.StringValue(cert.Common)
	config.SAN = types.StringValue(strings.Join(cert.SAN, ","))

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
