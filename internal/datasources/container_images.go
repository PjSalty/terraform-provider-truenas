package datasources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var _ datasource.DataSource = &ContainerImagesDataSource{}

// ContainerImagesDataSource lists the images a container can be created
// from, so a configuration does not have to hardcode a name and version
// that the registry rotates out within days.
type ContainerImagesDataSource struct {
	client *wsclient.Client
}

// ContainerImagesDataSourceModel describes the data source model.
type ContainerImagesDataSourceModel struct {
	ID           types.String          `tfsdk:"id"`
	NamePrefix   types.String          `tfsdk:"name_prefix"`
	Images       []ContainerImageModel `tfsdk:"images"`
	Names        types.List            `tfsdk:"names"`
	LatestByName types.Map             `tfsdk:"latest_by_name"`
}

// ContainerImageModel is one image and its published versions.
type ContainerImageModel struct {
	Name          types.String `tfsdk:"name"`
	Versions      types.List   `tfsdk:"versions"`
	LatestVersion types.String `tfsdk:"latest_version"`
}

// NewContainerImagesDataSource returns a new ContainerImagesDataSource factory.
func NewContainerImagesDataSource() datasource.DataSource {
	return &ContainerImagesDataSource{}
}

func (d *ContainerImagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_images"
}

func (d *ContainerImagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the images available to create LXC containers from on TrueNAS SCALE. " +
			"Containers are introduced in TrueNAS 26.0; this data source cannot be used against " +
			"25.10 or older. Reading it queries the upstream image registry, so it needs the " +
			"server to have a route to it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always `container_images`.",
				Computed:    true,
			},
			"name_prefix": schema.StringAttribute{
				Description: "Only return images whose name starts with this, for example `alpine:`. Empty returns everything.",
				Optional:    true,
			},
			"images": schema.ListNestedAttribute{
				Description: "Matching images, sorted by name.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Image name, for example `alpine:3.21:amd64:default`.",
							Computed:    true,
						},
						"versions": schema.ListAttribute{
							Description: "Published versions, oldest first, as the registry returns them.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"latest_version": schema.StringAttribute{
							Description: "The newest published version, which is the one least likely to age out.",
							Computed:    true,
						},
					},
				},
			},
			"names": schema.ListAttribute{
				Description: "Just the matching image names, sorted, for feeding a validation or a locals block.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"latest_by_name": schema.MapAttribute{
				Description: "Newest version keyed by image name. Versions are datestamped and the registry keeps only the most recent few, so resolving one here rather than hardcoding it stops a configuration rotting within days.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *ContainerImagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*wsclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *wsclient.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *ContainerImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ContainerImagesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	images, err := d.client.ListContainerImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Container Images",
			fmt.Sprintf("Could not list container images: %s", err),
		)
		return
	}

	prefix := config.NamePrefix.ValueString()
	matched := make([]truenas.ContainerImage, 0, len(images))
	for _, img := range images {
		if prefix != "" && !strings.HasPrefix(img.Name, prefix) {
			continue
		}
		matched = append(matched, img)
	}
	// The registry's order is not documented as stable, and an unstable
	// order would churn the list on every plan.
	sort.Slice(matched, func(i, j int) bool { return matched[i].Name < matched[j].Name })

	models := make([]ContainerImageModel, 0, len(matched))
	names := make([]string, 0, len(matched))
	latest := make(map[string]string, len(matched))

	for _, img := range matched {
		versions := make([]string, 0, len(img.Versions))
		for _, v := range img.Versions {
			versions = append(versions, v.Version)
		}
		// Converting a []string to a list of strings cannot fail, but the
		// diagnostics are appended rather than dropped: if that ever stops
		// being true, an error diagnostic fails the read on its own.
		versionList, diags := types.ListValueFrom(ctx, types.StringType, versions)
		resp.Diagnostics.Append(diags...)

		// The registry lists versions oldest-first, so the last entry is the
		// newest. An image with no versions is reported with an empty
		// latest_version rather than being dropped, so a configuration
		// referring to it fails on the empty value instead of on a missing
		// map key.
		newest := ""
		if len(versions) > 0 {
			newest = versions[len(versions)-1]
		}

		models = append(models, ContainerImageModel{
			Name:          types.StringValue(img.Name),
			Versions:      versionList,
			LatestVersion: types.StringValue(newest),
		})
		names = append(names, img.Name)
		latest[img.Name] = newest
	}

	nameList, diags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(diags...)
	latestMap, diags := types.MapValueFrom(ctx, types.StringType, latest)
	resp.Diagnostics.Append(diags...)

	config.ID = types.StringValue("container_images")
	config.Images = models
	config.Names = nameList
	config.LatestByName = latestMap

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
