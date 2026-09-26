// cloudsync_credential.go
//
// Polymorphic-provider handling:
//   - `provider_attributes_json` is a JSON-encoded string because the
//     upstream /cloudsync/credentials `provider` payload shape varies by
//     provider type (S3 has access_key_id/secret_access_key/endpoint, B2
//     has account_id/application_key, Azure has account_name/account_key,
//     etc.). Exposing it as a typed nested block would require per-provider
//     schemas and is impractical for a generic resource.
//   - Users pass e.g.
//     `provider_attributes_json = jsonencode({ access_key_id = "X",
//     secret_access_key = "Y" })`.
//   - The field is write-only from the API's perspective: TrueNAS masks
//     sensitive values (e.g. secret_access_key) in its own echo of the
//     credential, so mapResponseToModel never reconstructs it from the
//     server response, unlike cloud_backup.go's attributes_json.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                = &CloudSyncCredentialResource{}
	_ resource.ResourceWithImportState = &CloudSyncCredentialResource{}
)

// CloudSyncCredentialResource manages a TrueNAS cloud sync credential
// (S3, B2, Azure, GCS, Dropbox, etc.) used by cloud_sync and cloud_backup tasks.
type CloudSyncCredentialResource struct {
	client *wsclient.Client
}

// CloudSyncCredentialResourceModel describes the resource data model.
type CloudSyncCredentialResourceModel struct {
	ID                     types.String   `tfsdk:"id"`
	Name                   types.String   `tfsdk:"name"`
	ProviderType           types.String   `tfsdk:"provider_type"`
	ProviderAttributesJSON types.String   `tfsdk:"provider_attributes_json"`
	Timeouts               timeouts.Value `tfsdk:"timeouts"`
}

func NewCloudSyncCredentialResource() resource.Resource {
	return &CloudSyncCredentialResource{}
}

func (r *CloudSyncCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_credential"
}

func (r *CloudSyncCredentialResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a cloud sync credential (S3, B2, Azure, GCS, Dropbox, etc.) " +
			"on TrueNAS SCALE. Cloud sync credentials live under /cloudsync/credentials " +
			"and are distinct from keychain credentials (SSH). They are referenced by " +
			"numeric ID from truenas_cloud_sync and truenas_cloud_backup resources." + "\n\n" +
			"**Stability: GA.** Full `_basic` + `_disappears` + `_update` acceptance test triad verified live against TrueNAS SCALE 25.10.",
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
				Description: "The numeric ID of the cloud sync credential.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the credential.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"provider_type": schema.StringAttribute{
				Description: "The cloud provider type. One of: S3, B2, AZUREBLOB, " +
					"GOOGLE_CLOUD_STORAGE, DROPBOX, FTP, SFTP, HTTP, MEGA, " +
					"OPENSTACK_SWIFT, PCLOUD, WEBDAV, YANDEX, ONEDRIVE, " +
					"GOOGLE_DRIVE, BACKBLAZE_B2. Changing this forces replacement.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						// The discriminator values from the upstream cloud_sync_providers union,
						// identical across 25.10.0 through 27.0. BACKBLAZE_B2 is gone because no
						// version has ever accepted it; the provider is called B2.
						"AZUREBLOB", "B2", "BOX", "DROPBOX", "FTP", "GOOGLE_CLOUD_STORAGE", "GOOGLE_DRIVE",
						"GOOGLE_PHOTOS", "HTTP", "HUBIC", "MEGA", "ONEDRIVE", "OPENSTACK_SWIFT", "PCLOUD", "S3", "SFTP",
						"STORJ_IX", "WEBDAV", "YANDEX",
					),
				},
			},
			"provider_attributes_json": schema.StringAttribute{
				Description: "Provider-specific credential fields as a JSON object " +
					"(e.g. jsonencode({access_key_id = \"X\", secret_access_key = \"Y\"})). " +
					"The exact keys depend on provider_type. Write-only in practice: TrueNAS " +
					"masks sensitive fields (e.g. secret_access_key) in its own echo of the " +
					"credential, so mapResponseToModel below never reconstructs this attribute " +
					"from the server response the way cloud_sync.go/cloud_backup.go do for " +
					"their own (non-sensitive) attributes_json - it stays exactly what was " +
					"planned. Required, not Optional+Computed: nothing about it is actually " +
					"computed by the provider, so Required is the honest contract, and it also " +
					"avoids TestOptionalComputedHasUseStateForUnknown's UseStateForUnknown/Default " +
					"requirement for a field that has neither.",
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func (r *CloudSyncCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildProviderMap decodes the user-supplied JSON attributes and merges the
// top-level `type` key (from provider_type) so the payload matches the
// TrueNAS API expectation of `provider = {"type": "S3", ...}`.
func buildProviderMap(providerType, attributesJSON string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	if attributesJSON != "" {
		if err := json.Unmarshal([]byte(attributesJSON), &out); err != nil {
			return nil, fmt.Errorf("invalid provider_attributes_json: %w", err)
		}
	}
	out["type"] = providerType
	return out, nil
}

func (r *CloudSyncCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create CloudSyncCredential start")

	var plan CloudSyncCredentialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	providerMap, err := buildProviderMap(plan.ProviderType.ValueString(), plan.ProviderAttributesJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider_attributes_json", err.Error())
		return
	}

	createReq := &truenas.CloudSyncCredentialCreateRequest{
		Name:     plan.Name.ValueString(),
		Provider: providerMap,
	}

	tflog.Debug(ctx, "Creating cloud sync credential", map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"provider_type": plan.ProviderType.ValueString(),
	})

	cred, err := r.client.CreateCloudSyncCredential(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Cloud Sync Credential",
			fmt.Sprintf("Could not create cloud sync credential %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, cred, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create CloudSyncCredential success")
}

func (r *CloudSyncCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read CloudSyncCredential start")

	var state CloudSyncCredentialResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud sync credential ID: %s", err))
		return
	}

	cred, err := r.client.GetCloudSyncCredential(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Cloud Sync Credential",
			fmt.Sprintf("Could not read cloud sync credential %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, cred, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read CloudSyncCredential success")
}

func (r *CloudSyncCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update CloudSyncCredential start")

	var plan CloudSyncCredentialResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	var state CloudSyncCredentialResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud sync credential ID: %s", err))
		return
	}

	providerMap, err := buildProviderMap(plan.ProviderType.ValueString(), plan.ProviderAttributesJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider_attributes_json", err.Error())
		return
	}

	updateReq := &truenas.CloudSyncCredentialUpdateRequest{
		Name:     plan.Name.ValueString(),
		Provider: providerMap,
	}

	cred, err := r.client.UpdateCloudSyncCredential(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Cloud Sync Credential",
			fmt.Sprintf("Could not update cloud sync credential %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, cred, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update CloudSyncCredential success")
}

func (r *CloudSyncCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete CloudSyncCredential start")

	var state CloudSyncCredentialResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud sync credential ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting cloud sync credential", map[string]interface{}{"id": id})

	if err := r.client.DeleteCloudSyncCredential(ctx, id); err != nil {
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "Cloud sync credential already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Cloud Sync Credential",
			fmt.Sprintf("Could not delete cloud sync credential %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete CloudSyncCredential success")
}

func (r *CloudSyncCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Cloud sync credential ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToModel projects a CloudSyncCredential API response into the
// Terraform model, extracting the provider type. Deliberately does NOT
// touch model.ProviderAttributesJSON: TrueNAS masks sensitive credential
// fields (e.g. secret_access_key) in its own echo of the credential, so
// reconstructing that attribute from cred.Provider (as this function used
// to, via filterJSONByKeys/normalizeJSON - the same drift-suppression
// pattern cloud_sync.go and cloud_backup.go use for their own,
// non-sensitive attributes_json) can silently produce a value that
// differs from what was actually planned. When the plan's value for that
// field is already known (not depending on an as-yet-unapplied resource),
// Terraform requires the post-apply value to match it byte-for-byte or
// apply fails outright with "Provider produced inconsistent result after
// apply" - a hard crash on every Create/Update, not just a cosmetic
// diff. Leaving the field alone means Create/Update keep exactly what
// was planned (model IS the plan there) and Read/Import keep exactly
// what was already in state - this resource treats
// provider_attributes_json as write-only from the API's perspective,
// the same way user.go treats `password` and iscsi_auth.go treats
// `secret`/`peersecret` (see those resources' own ImportStateVerifyIgnore
// entries in importstate_verify_ignore_invariant_test.go).
func (r *CloudSyncCredentialResource) mapResponseToModel(_ context.Context, cred *truenas.CloudSyncCredential, model *CloudSyncCredentialResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(cred.ID))
	model.Name = types.StringValue(cred.Name)

	if providerType, ok := cred.Provider["type"].(string); ok && providerType != "" {
		model.ProviderType = types.StringValue(providerType)
	}
}
