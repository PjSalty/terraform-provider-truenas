// cloud_backup.go
//
// Polymorphic-attribute handling:
//   - `attributes_json` is exposed as a JSON-encoded string because the
//     upstream `attributes` payload shape varies by cloud provider (S3 adds
//     region/encryption/storage_class, Dropbox adds chunk_size, etc.) and
//     would be unwieldy as a nested block. Users pass
//     `attributes_json = jsonencode({ bucket = "my-bucket", region = "us-east-1" })`.
//   - We store the canonical form re-marshaled from the API response so
//     diffs are minimized as long as the user passes a stable key order.
package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &CloudBackupResource{}
	_ resource.ResourceWithImportState = &CloudBackupResource{}
	_ resource.ResourceWithModifyPlan  = &CloudBackupResource{}
)

type CloudBackupResource struct {
	client *client.Client
}

type CloudBackupResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	Description     types.String   `tfsdk:"description"`
	Path            types.String   `tfsdk:"path"`
	Credentials     types.Int64    `tfsdk:"credentials"`
	AttributesJSON  types.String   `tfsdk:"attributes_json"`
	PreScript       types.String   `tfsdk:"pre_script"`
	PostScript      types.String   `tfsdk:"post_script"`
	Snapshot        types.Bool     `tfsdk:"snapshot"`
	Include         types.List     `tfsdk:"include"`
	Exclude         types.List     `tfsdk:"exclude"`
	Args            types.String   `tfsdk:"args"`
	Enabled         types.Bool     `tfsdk:"enabled"`
	Password        types.String   `tfsdk:"password"`
	KeepLast        types.Int64    `tfsdk:"keep_last"`
	TransferSetting types.String   `tfsdk:"transfer_setting"`
	ScheduleMinute  types.String   `tfsdk:"schedule_minute"`
	ScheduleHour    types.String   `tfsdk:"schedule_hour"`
	ScheduleDom     types.String   `tfsdk:"schedule_dom"`
	ScheduleMonth   types.String   `tfsdk:"schedule_month"`
	ScheduleDow     types.String   `tfsdk:"schedule_dow"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

func NewCloudBackupResource() resource.Resource {
	return &CloudBackupResource{}
}

func (r *CloudBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_backup"
}

func (r *CloudBackupResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a cloud backup (restic) task on TrueNAS SCALE." + "\n\n" +
			"**Stability: Beta.** Create/read/update/destroy wire format verified against TrueNAS SCALE 25.10. Full end-to-end run with real cloud credentials has not been observed.",
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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Human-readable name for the backup task.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"path": schema.StringAttribute{
				Description: "Local path to back up (must begin with /mnt or /dev/zvol).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^/`),
						"must be an absolute path",
					),
				},
			},
			"credentials": schema.Int64Attribute{
				Description: "ID of the cloud credential to use for this task.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"attributes_json": schema.StringAttribute{
				Description: "Provider-specific attributes as a JSON object (e.g. jsonencode({bucket=\"b\", region=\"us-east-1\"})).",
				Required:    true,
			},
			"pre_script": schema.StringAttribute{
				Description: "Bash script to run immediately before each backup.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"post_script": schema.StringAttribute{
				Description: "Bash script to run immediately after each successful backup.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"snapshot": schema.BoolAttribute{
				Description: "Create a temporary snapshot of the dataset before each backup.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"include": schema.ListAttribute{
				Description: "Paths to pass to restic backup --include.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"exclude": schema.ListAttribute{
				Description: "Paths to pass to restic backup --exclude.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"args": schema.StringAttribute{
				Description: "Additional args (slated for removal upstream).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether this task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"password": schema.StringAttribute{
				Description: "Password for the remote restic repository.",
				Required:    true,
				Sensitive:   true,
			},
			"keep_last": schema.Int64Attribute{
				Description: "How many of the most recent backup snapshots to keep after each backup (1-9999).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 9999),
				},
			},
			"transfer_setting": schema.StringAttribute{
				Description: "One of DEFAULT, PERFORMANCE, FAST_STORAGE.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DEFAULT"),
				Validators: []validator.String{
					stringvalidator.OneOf("DEFAULT", "PERFORMANCE", "FAST_STORAGE"),
				},
			},
			"schedule_minute": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("00"),
			},
			"schedule_hour": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("*"),
			},
			"schedule_dom": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("*"),
			},
			"schedule_month": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("*"),
			},
			"schedule_dow": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("*"),
			},
		},
	}
}

func (r *CloudBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CloudBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create CloudBackup start")

	var plan CloudBackupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs, err := normalizeJSON(plan.AttributesJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid attributes_json", fmt.Sprintf("%s", err))
		return
	}

	snapshot := plan.Snapshot.ValueBool()
	enabled := plan.Enabled.ValueBool()

	createReq := &client.CloudBackupCreateRequest{
		Description:     plan.Description.ValueString(),
		Path:            plan.Path.ValueString(),
		Credentials:     int(plan.Credentials.ValueInt64()),
		Attributes:      attrs,
		PreScript:       plan.PreScript.ValueString(),
		PostScript:      plan.PostScript.ValueString(),
		Snapshot:        &snapshot,
		Args:            plan.Args.ValueString(),
		Enabled:         &enabled,
		Password:        plan.Password.ValueString(),
		KeepLast:        int(plan.KeepLast.ValueInt64()),
		TransferSetting: plan.TransferSetting.ValueString(),
		Schedule: &client.CloudBackupSchedule{
			Minute: plan.ScheduleMinute.ValueString(),
			Hour:   plan.ScheduleHour.ValueString(),
			Dom:    plan.ScheduleDom.ValueString(),
			Month:  plan.ScheduleMonth.ValueString(),
			Dow:    plan.ScheduleDow.ValueString(),
		},
	}

	if !plan.Include.IsNull() && !plan.Include.IsUnknown() {
		var include []string
		resp.Diagnostics.Append(plan.Include.ElementsAs(ctx, &include, false)...)
		createReq.Include = include
	}
	if !plan.Exclude.IsNull() && !plan.Exclude.IsUnknown() {
		var exclude []string
		resp.Diagnostics.Append(plan.Exclude.ElementsAs(ctx, &exclude, false)...)
		createReq.Exclude = exclude
	}

	tflog.Debug(ctx, "Creating cloud backup task")

	cb, err := r.client.CreateCloudBackup(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Cloud Backup", fmt.Sprintf("Could not create cloud backup: %s", err))
		return
	}

	r.mapResponseToModel(ctx, cb, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create CloudBackup success")
}

func (r *CloudBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read CloudBackup start")

	var state CloudBackupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud backup ID: %s", err))
		return
	}

	cb, err := r.client.GetCloudBackup(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Cloud Backup", fmt.Sprintf("Could not read cloud backup %d: %s", id, err))
		return
	}

	r.mapResponseToModel(ctx, cb, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read CloudBackup success")
}

func (r *CloudBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update CloudBackup start")

	var plan CloudBackupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CloudBackupResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud backup ID: %s", err))
		return
	}

	attrs, err := normalizeJSON(plan.AttributesJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid attributes_json", fmt.Sprintf("%s", err))
		return
	}

	description := plan.Description.ValueString()
	backupPath := plan.Path.ValueString()
	creds := int(plan.Credentials.ValueInt64())
	preScript := plan.PreScript.ValueString()
	postScript := plan.PostScript.ValueString()
	snapshot := plan.Snapshot.ValueBool()
	args := plan.Args.ValueString()
	enabled := plan.Enabled.ValueBool()
	password := plan.Password.ValueString()
	keepLast := int(plan.KeepLast.ValueInt64())
	transferSetting := plan.TransferSetting.ValueString()

	updateReq := &client.CloudBackupUpdateRequest{
		Description:     &description,
		Path:            &backupPath,
		Credentials:     &creds,
		Attributes:      attrs,
		PreScript:       &preScript,
		PostScript:      &postScript,
		Snapshot:        &snapshot,
		Args:            &args,
		Enabled:         &enabled,
		Password:        &password,
		KeepLast:        &keepLast,
		TransferSetting: &transferSetting,
		Schedule: &client.CloudBackupSchedule{
			Minute: plan.ScheduleMinute.ValueString(),
			Hour:   plan.ScheduleHour.ValueString(),
			Dom:    plan.ScheduleDom.ValueString(),
			Month:  plan.ScheduleMonth.ValueString(),
			Dow:    plan.ScheduleDow.ValueString(),
		},
	}

	if !plan.Include.IsNull() && !plan.Include.IsUnknown() {
		var include []string
		resp.Diagnostics.Append(plan.Include.ElementsAs(ctx, &include, false)...)
		updateReq.Include = &include
	}
	if !plan.Exclude.IsNull() && !plan.Exclude.IsUnknown() {
		var exclude []string
		resp.Diagnostics.Append(plan.Exclude.ElementsAs(ctx, &exclude, false)...)
		updateReq.Exclude = &exclude
	}

	cb, err := r.client.UpdateCloudBackup(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Cloud Backup", fmt.Sprintf("Could not update cloud backup %d: %s", id, err))
		return
	}

	r.mapResponseToModel(ctx, cb, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update CloudBackup success")
}

func (r *CloudBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete CloudBackup start")

	var state CloudBackupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud backup ID: %s", err))
		return
	}

	if err := r.client.DeleteCloudBackup(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Cloud backup task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting Cloud Backup", fmt.Sprintf("Could not delete cloud backup %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete CloudBackup success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *CloudBackupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_cloud_backup")
}

func (r *CloudBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Cloud backup ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), types.StringValue(""))...)
}

func (r *CloudBackupResource) mapResponseToModel(ctx context.Context, cb *client.CloudBackup, model *CloudBackupResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(cb.ID))
	model.Description = types.StringValue(cb.Description)
	model.Path = types.StringValue(cb.Path)
	// credentials may be an object { id, name, provider } or an int — try both.
	credsID := extractCredentialsID(cb.Credentials)
	model.Credentials = types.Int64Value(int64(credsID))
	if len(cb.Attributes) > 0 {
		// Preserve the user-supplied subset: TrueNAS fills in server-side
		// defaults that the user never wrote, which otherwise triggers
		// "inconsistent result after apply" / phantom drift.
		prior := model.AttributesJSON.ValueString()
		if filtered, err := filterJSONByKeys(string(cb.Attributes), prior); err == nil {
			// filtered is produced by json.Marshal of a decoded map, so
			// normalizeJSON cannot fail here.
			canon, _ := normalizeJSON(filtered)
			model.AttributesJSON = types.StringValue(string(canon))
		} else {
			// filterJSONByKeys only errors on malformed cb.Attributes.
			model.AttributesJSON = types.StringValue(string(cb.Attributes))
		}
	}
	model.PreScript = types.StringValue(cb.PreScript)
	model.PostScript = types.StringValue(cb.PostScript)
	model.Snapshot = types.BoolValue(cb.Snapshot)
	model.Args = types.StringValue(cb.Args)
	model.Enabled = types.BoolValue(cb.Enabled)
	// Preserve password: do not overwrite user value with API (which is redacted).
	if model.Password.IsNull() || model.Password.IsUnknown() {
		model.Password = types.StringValue("")
	}
	model.KeepLast = types.Int64Value(int64(cb.KeepLast))
	if cb.TransferSetting != "" {
		model.TransferSetting = types.StringValue(cb.TransferSetting)
	}
	model.ScheduleMinute = types.StringValue(cb.Schedule.Minute)
	model.ScheduleHour = types.StringValue(cb.Schedule.Hour)
	model.ScheduleDom = types.StringValue(cb.Schedule.Dom)
	model.ScheduleMonth = types.StringValue(cb.Schedule.Month)
	model.ScheduleDow = types.StringValue(cb.Schedule.Dow)

	if cb.Include != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, cb.Include)
		if !diags.HasError() {
			model.Include = list
		}
	}
	if cb.Exclude != nil {
		list, diags := types.ListValueFrom(ctx, types.StringType, cb.Exclude)
		if !diags.HasError() {
			model.Exclude = list
		}
	}

	_ = int64default.StaticInt64 // keep import
}

// extractCredentialsID handles both int and expanded object shapes from the API.
func extractCredentialsID(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil {
		return asInt
	}
	var asObj struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(raw, &asObj); err == nil {
		return asObj.ID
	}
	return 0
}

// normalizeJSON parses and re-marshals JSON for stable comparison.
func normalizeJSON(raw string) (json.RawMessage, error) {
	if raw == "" {
		return json.RawMessage("{}"), nil
	}
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	// json.Marshal cannot fail on a value that came from json.Decode.
	out, _ := json.Marshal(v)
	return json.RawMessage(out), nil
}

// stripJSONNulls recursively removes map entries whose value is explicitly null.
// TrueNAS often echoes back payloads with server-side defaults set to null
// (e.g. ACL entries grow a "who": null field after apply). That asymmetry
// breaks Terraform's "result must equal plan" invariant on Create/Update
// even though the user never wrote the null. Stripping them here makes
// round-trips deterministic.
func stripJSONNulls(raw string) (string, error) {
	if raw == "" {
		return raw, nil
	}
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	cleaned := stripNullsFromValue(v)
	// json.Marshal cannot fail on a value derived from json.Decode.
	out, _ := json.Marshal(cleaned)
	return string(out), nil
}

func stripNullsFromValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			if val == nil {
				continue
			}
			out[k] = stripNullsFromValue(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = stripNullsFromValue(val)
		}
		return out
	default:
		return v
	}
}

// filterJSONByKeys returns a JSON-serialized copy of `server` that contains only
// the top-level keys also present in `reference`, with values taken from `server`.
// This is the standard solution for "server fills in defaults the user
// didn't write": we persist only the keys the user opted into, so Terraform's
// plan (user config subset) equals state (same subset) and no phantom drift
// appears. If reference is empty (e.g. first import), returns the full server
// object in canonical form.
func filterJSONByKeys(server, reference string) (string, error) {
	if server == "" {
		return "{}", nil
	}

	var serverMap map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(server)))
	dec.UseNumber()
	if err := dec.Decode(&serverMap); err != nil {
		return "", fmt.Errorf("invalid server JSON: %w", err)
	}

	// No reference → return full server object in canonical form.
	// json.Marshal cannot fail on a value that came from json.Decode.
	if reference == "" {
		out, _ := json.Marshal(serverMap)
		return string(out), nil
	}

	var refMap map[string]interface{}
	rdec := json.NewDecoder(bytes.NewReader([]byte(reference)))
	rdec.UseNumber()
	// Intentional fallback: if the reference isn't a JSON object (e.g. it's
	// a raw array or null), fall through to "no filter" and emit the full
	// server object. This is the drift-suppression contract for first-import
	// paths where the caller cannot yet know the key-set shape. Returning
	// nil here is NOT error-swallowing — it's documented behavior.
	//nolint:nilerr // intentional decode-fallback; see doc comment above
	if err := rdec.Decode(&refMap); err != nil {
		out, _ := json.Marshal(serverMap)
		return string(out), nil
	}

	filtered := make(map[string]interface{}, len(refMap))
	for k := range refMap {
		if v, ok := serverMap[k]; ok {
			filtered[k] = v
		}
	}

	out, _ := json.Marshal(filtered)
	return string(out), nil
}
