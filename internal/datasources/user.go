package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &UserDataSource{}

// UserDataSource provides information about a TrueNAS user.
type UserDataSource struct {
	client *client.Client
}

// UserDataSourceModel describes the data source model.
type UserDataSourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	FullName types.String `tfsdk:"full_name"`
	UID      types.Int64  `tfsdk:"uid"`
	GID      types.Int64  `tfsdk:"gid"`
	Home     types.String `tfsdk:"home"`
	Shell    types.String `tfsdk:"shell"`
	Locked   types.Bool   `tfsdk:"locked"`
	SMB      types.Bool   `tfsdk:"smb"`
	Email    types.String `tfsdk:"email"`
	Builtin  types.Bool   `tfsdk:"builtin"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a user on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The user ID.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username to look up.",
				Required:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "The full name of the user.",
				Computed:    true,
			},
			"uid": schema.Int64Attribute{
				Description: "The UNIX UID.",
				Computed:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "The primary group GID.",
				Computed:    true,
			},
			"home": schema.StringAttribute{
				Description: "The home directory.",
				Computed:    true,
			},
			"shell": schema.StringAttribute{
				Description: "The login shell.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the user account is locked.",
				Computed:    true,
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the user has SMB access.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The user email address.",
				Computed:    true,
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system user.",
				Computed:    true,
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := config.Username.ValueString()

	user, err := d.client.GetUserByUsername(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User",
			fmt.Sprintf("Could not find user %q: %s", username, err),
		)
		return
	}

	config.ID = types.Int64Value(int64(user.ID))
	config.Username = types.StringValue(user.Username)
	config.FullName = types.StringValue(user.FullName)
	config.UID = types.Int64Value(int64(user.UID))
	config.Home = types.StringValue(user.Home)
	config.Shell = types.StringValue(user.Shell)
	config.Locked = types.BoolValue(user.Locked)
	config.SMB = types.BoolValue(user.SMB)
	config.Builtin = types.BoolValue(user.Builtin)

	config.GID = types.Int64Value(int64(user.Group.GID))

	if user.Email != nil {
		config.Email = types.StringValue(*user.Email)
	} else {
		config.Email = types.StringValue("")
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
