package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	"github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
	"github.com/PjSalty/terraform-provider-truenas/internal/resourcevalidators"
)

var (
	_ resource.Resource                     = &CertificateResource{}
	_ resource.ResourceWithImportState      = &CertificateResource{}
	_ resource.ResourceWithModifyPlan       = &CertificateResource{}
	_ resource.ResourceWithConfigValidators = &CertificateResource{}
)

// CertificateResource manages a TrueNAS TLS certificate.
type CertificateResource struct {
	client *client.Client
}

// CertificateResourceModel describes the resource data model.
type CertificateResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	CreateType         types.String   `tfsdk:"create_type"`
	Certificate        types.String   `tfsdk:"certificate"`
	Privatekey         types.String   `tfsdk:"privatekey"`
	KeyType            types.String   `tfsdk:"key_type"`
	KeyLength          types.Int64    `tfsdk:"key_length"`
	DigestAlgorithm    types.String   `tfsdk:"digest_algorithm"`
	Lifetime           types.Int64    `tfsdk:"lifetime"`
	Country            types.String   `tfsdk:"country"`
	State              types.String   `tfsdk:"state"`
	City               types.String   `tfsdk:"city"`
	Organization       types.String   `tfsdk:"organization"`
	OrganizationalUnit types.String   `tfsdk:"organizational_unit"`
	Email              types.String   `tfsdk:"email"`
	Common             types.String   `tfsdk:"common"`
	SAN                types.List     `tfsdk:"san"`
	DN                 types.String   `tfsdk:"dn"`
	From               types.String   `tfsdk:"from"`
	Until              types.String   `tfsdk:"until"`
	Expired            types.Bool     `tfsdk:"expired"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

func NewCertificateResource() resource.Resource {
	return &CertificateResource{}
}

func (r *CertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (r *CertificateResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a TLS certificate on TrueNAS SCALE. " +
		"Default timeouts: 20m create (ACME/CSR signing can be slow), 10m update/delete.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the certificate.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the certificate.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"create_type": schema.StringAttribute{
				Description: "The certificate creation type: CERTIFICATE_CREATE_IMPORTED, CERTIFICATE_CREATE_CSR, CERTIFICATE_CREATE_IMPORTED_CSR, or CERTIFICATE_CREATE_ACME.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"CERTIFICATE_CREATE_IMPORTED",
						"CERTIFICATE_CREATE_CSR",
						"CERTIFICATE_CREATE_IMPORTED_CSR",
						"CERTIFICATE_CREATE_ACME",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate": schema.StringAttribute{
				Description: "The PEM-encoded certificate data. Required for CERTIFICATE_CREATE_IMPORTED.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// UseStateForUnknown prevents Unknown ("known after apply")
					// from being compared to a concrete state value when the
					// user omits this Optional+Computed attribute from HCL on
					// a later apply. PEMEquivalent then runs to suppress
					// cosmetic normalization from the server on read-back
					// (CRLF→LF, base64 rewrap, trailing whitespace). Only
					// after both of those has RequiresReplace a chance to
					// see a genuine byte-level change that actually warrants
					// destroy+create.
					stringplanmodifier.UseStateForUnknown(),
					planmodifiers.PEMEquivalent(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"privatekey": schema.StringAttribute{
				Description: "The PEM-encoded private key. Required for CERTIFICATE_CREATE_IMPORTED.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.PEMEquivalent(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_type": schema.StringAttribute{
				Description: "The key type: RSA or EC.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("RSA", "EC"),
				},
				PlanModifiers: []planmodifier.String{
					// UseStateForUnknown MUST run before RequiresReplace: for
					// Optional+Computed attributes the framework otherwise
					// marks the plan value as Unknown ("known after apply")
					// when the user omits the attribute from HCL, and
					// RequiresReplace then compares Unknown to the state
					// value and falsely forces a destroy+create.
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_length": schema.Int64Attribute{
				Description: "The key length in bits (1024, 2048, 4096).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1024, 2048, 4096),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"digest_algorithm": schema.StringAttribute{
				Description: "The digest algorithm (e.g., SHA256, SHA384).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("SHA224", "SHA256", "SHA384", "SHA512"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"lifetime": schema.Int64Attribute{
				Description: "The certificate lifetime in days (1-36500).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 36500),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"country": schema.StringAttribute{
				Description: "The certificate country (C). Two-letter ISO 3166 code.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 2),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Description: "The certificate state/province (ST).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"city": schema.StringAttribute{
				Description: "The certificate city/locality (L).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "The certificate organization (O).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organizational_unit": schema.StringAttribute{
				Description: "The certificate organizational unit (OU).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Description: "The certificate email address.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(253),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"common": schema.StringAttribute{
				Description: "The common name (CN) of the certificate.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(253),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"san": schema.ListAttribute{
				Description: "Subject alternative names.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"dn": schema.StringAttribute{
				Description: "The full distinguished name.",
				Computed:    true,
			},
			"from": schema.StringAttribute{
				Description: "The certificate valid-from date.",
				Computed:    true,
			},
			"until": schema.StringAttribute{
				Description: "The certificate valid-until date.",
				Computed:    true,
			},
			"expired": schema.BoolAttribute{
				Description: "Whether the certificate has expired.",
				Computed:    true,
			},
		},
	}
}

// ConfigValidators enforces cross-attribute rules at config-validation
// time, before any network round-trip. Today this holds the
// create_type → {certificate, privatekey} requirement: if the user
// selects CERTIFICATE_CREATE_IMPORTED in HCL, both PEM attributes must
// be set. Other create_type values (CSR, IMPORTED_CSR, ACME) have their
// own requirements but TrueNAS enforces those server-side during the
// certificate.create job, which surfaces them as a normal API error;
// IMPORTED is the one value where a missing PEM causes a cryptic
// "job failed" rather than an actionable diagnostic, so we catch it
// client-side here.
func (r *CertificateResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidators.RequiredWhenEqual(
			"create_type",
			"CERTIFICATE_CREATE_IMPORTED",
			[]string{"certificate", "privatekey"},
		),
	}
}

func (r *CertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Certificate start")

	var plan CertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CertificateCreateRequest{
		Name:       plan.Name.ValueString(),
		CreateType: plan.CreateType.ValueString(),
	}

	if !plan.Certificate.IsNull() {
		createReq.CertificateData = plan.Certificate.ValueString()
	}
	if !plan.Privatekey.IsNull() {
		createReq.Privatekey = plan.Privatekey.ValueString()
	}
	if !plan.KeyType.IsNull() {
		createReq.KeyType = plan.KeyType.ValueString()
	}
	if !plan.KeyLength.IsNull() {
		createReq.KeyLength = int(plan.KeyLength.ValueInt64())
	}
	if !plan.DigestAlgorithm.IsNull() {
		createReq.DigestAlgorithm = plan.DigestAlgorithm.ValueString()
	}
	if !plan.Country.IsNull() {
		createReq.Country = plan.Country.ValueString()
	}
	if !plan.State.IsNull() {
		createReq.State = plan.State.ValueString()
	}
	if !plan.City.IsNull() {
		createReq.City = plan.City.ValueString()
	}
	if !plan.Organization.IsNull() {
		createReq.Organization = plan.Organization.ValueString()
	}
	if !plan.OrganizationalUnit.IsNull() {
		createReq.OrganizationalUnit = plan.OrganizationalUnit.ValueString()
	}
	if !plan.Email.IsNull() {
		createReq.Email = plan.Email.ValueString()
	}
	if !plan.Common.IsNull() {
		createReq.Common = plan.Common.ValueString()
	}
	if !plan.SAN.IsNull() && !plan.SAN.IsUnknown() {
		var sans []string
		resp.Diagnostics.Append(plan.SAN.ElementsAs(ctx, &sans, false)...)
		createReq.SAN = sans
	}

	tflog.Debug(ctx, "Creating certificate", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	cert, err := r.client.CreateCertificate(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Certificate",
			fmt.Sprintf("Could not create certificate %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, cert, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Certificate success")
}

func (r *CertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Certificate start")

	var state CertificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse certificate ID: %s", err))
		return
	}

	cert, err := r.client.GetCertificate(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Certificate",
			fmt.Sprintf("Could not read certificate %d: %s", id, err),
		)
		return
	}

	// Preserve the privatekey from state since the API masks it
	privatekey := state.Privatekey
	r.mapResponseToModel(ctx, cert, &state)
	state.Privatekey = privatekey

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Certificate success")
}

func (r *CertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Certificate start")

	var plan CertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CertificateResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse certificate ID: %s", err))
		return
	}

	updateReq := &client.CertificateUpdateRequest{
		Name: plan.Name.ValueString(),
	}

	cert, err := r.client.UpdateCertificate(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Certificate",
			fmt.Sprintf("Could not update certificate %d: %s", id, err),
		)
		return
	}

	// Preserve the privatekey from state since the API masks it
	privatekey := state.Privatekey
	r.mapResponseToModel(ctx, cert, &plan)
	plan.Privatekey = privatekey

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Certificate success")
}

func (r *CertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Certificate start")

	var state CertificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse certificate ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting certificate", map[string]interface{}{"id": id})

	err = r.client.DeleteCertificate(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Certificate already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Certificate",
			fmt.Sprintf("Could not delete certificate %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Certificate success")
}

// ModifyPlan enforces certificate cross-attribute constraints:
//
//   - create_type=CERTIFICATE_CREATE_IMPORTED requires `certificate` and
//     `privatekey` to be set (that's the whole point of the IMPORTED type).
//   - create_type=CERTIFICATE_CREATE_CSR requires `common` or at least one
//     SAN entry (you can't request a cert without an identity).
func (r *CertificateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_certificate")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config CertificateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if config.CreateType.IsNull() || config.CreateType.IsUnknown() {
		return
	}
	createType := config.CreateType.ValueString()

	certSet := !config.Certificate.IsNull() && !config.Certificate.IsUnknown() && config.Certificate.ValueString() != ""
	pkSet := !config.Privatekey.IsNull() && !config.Privatekey.IsUnknown() && config.Privatekey.ValueString() != ""
	commonSet := !config.Common.IsNull() && !config.Common.IsUnknown() && config.Common.ValueString() != ""
	sanSet := !config.SAN.IsNull() && !config.SAN.IsUnknown() && len(config.SAN.Elements()) > 0

	switch createType {
	case "CERTIFICATE_CREATE_IMPORTED":
		if !certSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("certificate"),
				"Missing certificate",
				"create_type=CERTIFICATE_CREATE_IMPORTED requires `certificate` to be set to a PEM-encoded certificate.",
			)
		}
		if !pkSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("privatekey"),
				"Missing privatekey",
				"create_type=CERTIFICATE_CREATE_IMPORTED requires `privatekey` to be set to a PEM-encoded private key.",
			)
		}
	case "CERTIFICATE_CREATE_CSR":
		if !commonSet && !sanSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("common"),
				"Missing identity",
				"create_type=CERTIFICATE_CREATE_CSR requires either `common` (CN) or at least one SAN entry.",
			)
		}
	}
}

func (r *CertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Certificate ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("create_type"), types.StringValue("CERTIFICATE_CREATE_IMPORTED"))...)
}

func (r *CertificateResource) mapResponseToModel(ctx context.Context, cert *client.Certificate, model *CertificateResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(cert.ID))
	model.Name = types.StringValue(cert.Name)
	model.Certificate = types.StringValue(cert.CertificateData)
	model.KeyType = types.StringValue(cert.KeyType)
	model.KeyLength = types.Int64Value(int64(cert.KeyLength))
	model.DigestAlgorithm = types.StringValue(cert.DigestAlgorithm)
	model.Lifetime = types.Int64Value(int64(cert.Lifetime))
	model.DN = types.StringValue(cert.DN)
	model.From = types.StringValue(cert.From)
	model.Until = types.StringValue(cert.Until)
	model.Expired = types.BoolValue(cert.Expired)

	// These may be empty strings from the API; set them appropriately
	if cert.Country != "" {
		model.Country = types.StringValue(cert.Country)
	} else {
		model.Country = types.StringValue("")
	}
	if cert.State != "" {
		model.State = types.StringValue(cert.State)
	} else {
		model.State = types.StringValue("")
	}
	if cert.City != "" {
		model.City = types.StringValue(cert.City)
	} else {
		model.City = types.StringValue("")
	}
	if cert.Organization != "" {
		model.Organization = types.StringValue(cert.Organization)
	} else {
		model.Organization = types.StringValue("")
	}
	if cert.OrganizationalUnit != "" {
		model.OrganizationalUnit = types.StringValue(cert.OrganizationalUnit)
	} else {
		model.OrganizationalUnit = types.StringValue("")
	}
	if cert.Email != "" {
		model.Email = types.StringValue(cert.Email)
	} else {
		model.Email = types.StringValue("")
	}
	if cert.Common != "" {
		model.Common = types.StringValue(cert.Common)
	} else {
		model.Common = types.StringValue("")
	}

	// SAN from API
	sanValues := make([]string, 0, len(cert.SAN))
	sanValues = append(sanValues, cert.SAN...)
	model.SAN, _ = types.ListValueFrom(ctx, types.StringType, sanValues)
}
