package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/customtypes"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	"github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
	"github.com/PjSalty/terraform-provider-truenas/internal/resourcevalidators"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                     = &CertificateResource{}
	_ resource.ResourceWithImportState      = &CertificateResource{}
	_ resource.ResourceWithModifyPlan       = &CertificateResource{}
	_ resource.ResourceWithConfigValidators = &CertificateResource{}
)

// CertificateResource manages a TrueNAS TLS certificate.
type CertificateResource struct {
	client *wsclient.Client
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
	TOS                types.Bool     `tfsdk:"tos"`
	CSRID              types.Int64    `tfsdk:"csr_id"`
	AcmeDirectoryURI   types.String   `tfsdk:"acme_directory_uri"`
	RenewDays          types.Int64    `tfsdk:"renew_days"`
	DNSMapping         types.Map      `tfsdk:"dns_mapping"`
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
				Description: "The key length in bits. Required when key_type is RSA. " +
					"1024 was never accepted: upstream declares Literal[2048, 4096] on " +
					"every supported release, so offering it only moved the failure to " +
					"apply time.",
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					int64validator.OneOf(2048, 4096),
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
				Description: "Subject alternative names, as bare values such as " +
					"`example.com`. TrueNAS reads them back from the parsed certificate " +
					"with their general-name kind attached (`DNS:example.com`), which is " +
					"compared as equal rather than as drift.",
				Optional:    true,
				Computed:    true,
				ElementType: customtypes.SANEntryType{},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},

			// ACME. All five are rejected outright on any other create_type,
			// and all but renew_days are required for this one. ModifyPlan
			// enforces both directions; see the CERTIFICATE_CREATE_ACME case
			// there for which upstream check each one answers.
			"tos": schema.BoolAttribute{
				Description: "CERTIFICATE_CREATE_ACME only, and required for it. Accept the " +
					"ACME provider's terms of service.",
				Optional: true,
			},
			"csr_id": schema.Int64Attribute{
				Description: "CERTIFICATE_CREATE_ACME only, and required for it. ID of an " +
					"existing certificate signing request to satisfy, typically another " +
					"`truenas_certificate` created with CERTIFICATE_CREATE_CSR.",
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"acme_directory_uri": schema.StringAttribute{
				Description: "CERTIFICATE_CREATE_ACME only, and required for it. The ACME " +
					"directory URI, for example " +
					"`https://acme-v02.api.letsencrypt.org/directory`.",
				Optional: true,
			},
			"renew_days": schema.Int64Attribute{
				Description: "CERTIFICATE_CREATE_ACME only. Days before expiry to attempt " +
					"renewal, between 1 and 30. Defaults to 10 when unset.",
				Optional: true,
				Validators: []validator.Int64{
					int64validator.Between(1, 30),
				},
			},
			"dns_mapping": schema.MapAttribute{
				Description: "CERTIFICATE_CREATE_ACME only, and required for it. Maps each " +
					"domain in the CSR to the ID of a `truenas_acme_dns_authenticator` that " +
					"can complete the DNS-01 challenge for it. The keys are the CSR's " +
					"`common` and `san` values as written there, and every one of them must " +
					"appear: TrueNAS refuses the request naming any domain it cannot " +
					"authenticate, and refuses a key that is not in the CSR.",
				Optional:    true,
				ElementType: types.Int64Type,
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

func (r *CertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Certificate start")

	var plan CertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &truenas.CertificateCreateRequest{
		Name:       plan.Name.ValueString(),
		CreateType: plan.CreateType.ValueString(),
	}

	if !plan.Certificate.IsNull() && !plan.Certificate.IsUnknown() {
		createReq.CertificateData = plan.Certificate.ValueString()
	}
	if !plan.Privatekey.IsNull() && !plan.Privatekey.IsUnknown() {
		createReq.Privatekey = plan.Privatekey.ValueString()
	}
	if !plan.KeyType.IsNull() && !plan.KeyType.IsUnknown() {
		createReq.KeyType = plan.KeyType.ValueString()
	}
	if !plan.KeyLength.IsNull() && !plan.KeyLength.IsUnknown() {
		createReq.KeyLength = int(plan.KeyLength.ValueInt64())
	}
	if !plan.DigestAlgorithm.IsNull() && !plan.DigestAlgorithm.IsUnknown() {
		createReq.DigestAlgorithm = plan.DigestAlgorithm.ValueString()
	}
	if !plan.Country.IsNull() && !plan.Country.IsUnknown() {
		createReq.Country = plan.Country.ValueString()
	}
	if !plan.State.IsNull() && !plan.State.IsUnknown() {
		createReq.State = plan.State.ValueString()
	}
	if !plan.City.IsNull() && !plan.City.IsUnknown() {
		createReq.City = plan.City.ValueString()
	}
	if !plan.Organization.IsNull() && !plan.Organization.IsUnknown() {
		createReq.Organization = plan.Organization.ValueString()
	}
	if !plan.OrganizationalUnit.IsNull() && !plan.OrganizationalUnit.IsUnknown() {
		createReq.OrganizationalUnit = plan.OrganizationalUnit.ValueString()
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		createReq.Email = plan.Email.ValueString()
	}
	if !plan.Common.IsNull() && !plan.Common.IsUnknown() {
		createReq.Common = plan.Common.ValueString()
	}
	if !plan.SAN.IsNull() && !plan.SAN.IsUnknown() {
		var sans []string
		resp.Diagnostics.Append(plan.SAN.ElementsAs(ctx, &sans, false)...)
		// Send bare values. The practitioner may have written the kind
		// explicitly after reading it back, and the server rejects a doubled
		// prefix.
		for i, v := range sans {
			sans[i] = customtypes.StripSANPrefix(v)
		}
		createReq.SAN = sans
	}

	// ACME. All five are required for that create_type and rejected on the
	// others, which ModifyPlan checks first, so sending them unconditionally
	// when set is safe.
	if !plan.TOS.IsNull() && !plan.TOS.IsUnknown() {
		v := plan.TOS.ValueBool()
		createReq.TOS = &v
	}
	if !plan.CSRID.IsNull() && !plan.CSRID.IsUnknown() {
		v := int(plan.CSRID.ValueInt64())
		createReq.CSRID = &v
	}
	if !plan.AcmeDirectoryURI.IsNull() && !plan.AcmeDirectoryURI.IsUnknown() {
		createReq.AcmeDirectoryURI = plan.AcmeDirectoryURI.ValueString()
	}
	if !plan.RenewDays.IsNull() && !plan.RenewDays.IsUnknown() {
		v := int(plan.RenewDays.ValueInt64())
		createReq.RenewDays = &v
	}
	if !plan.DNSMapping.IsNull() && !plan.DNSMapping.IsUnknown() {
		mapping := map[string]int64{}
		resp.Diagnostics.Append(plan.DNSMapping.ElementsAs(ctx, &mapping, false)...)
		createReq.DNSMapping = make(map[string]int, len(mapping))
		for k, v := range mapping {
			createReq.DNSMapping[k] = int(v)
		}
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

	resp.Diagnostics.Append(r.mapResponseToModel(ctx, cert, &plan)...)

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
		if wsclient.IsNotFound(err) {
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
	resp.Diagnostics.Append(r.mapResponseToModel(ctx, cert, &state)...)
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

	updateReq := &truenas.CertificateUpdateRequest{
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
	resp.Diagnostics.Append(r.mapResponseToModel(ctx, cert, &plan)...)
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
		if wsclient.IsNotFound(err) {
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
		// The server requires a non-empty san outright, not "common or san".
		// CertificateCreateCSRArgs declares san with min_length=1, so a CSR
		// with only a common name failed at apply while planning clean:
		//
		//	[EINVAL] certificate_create_csr.san: List should have at least 1
		//	item after validation, not 0
		if !sanSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("san"),
				"Missing SAN",
				"create_type=CERTIFICATE_CREATE_CSR requires at least one `san` entry. "+
					"A common name alone is not enough: TrueNAS validates the request "+
					"against a model that requires a non-empty san.",
			)
		}
		// An RSA key needs key_length with it:
		//
		//	if key_type != 'EC':
		//	    if not data.get('key_length'):
		//	        verrors.add(f'{schema}.key_length',
		//	                    'RSA-based keys require an entry in this field.')
		//
		// key_type and digest_algorithm are checked in the same block and are
		// deliberately NOT required here: both carry a model default ('RSA',
		// 'SHA256') that is filled in before the check reads them, so demanding
		// them would refuse configurations the server accepts. key_length is
		// the only one of the three that defaults to null.
		keyType := "RSA"
		if !config.KeyType.IsNull() && !config.KeyType.IsUnknown() && config.KeyType.ValueString() != "" {
			keyType = config.KeyType.ValueString()
		}
		if keyType != "EC" && (config.KeyLength.IsNull() || config.KeyLength.IsUnknown()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("key_length"),
				"Missing key_length",
				"create_type=CERTIFICATE_CREATE_CSR with an RSA key requires `key_length` "+
					"(2048 or 4096). Set `key_type = \"EC\"` if you did not mean RSA.",
			)
		}
	case "CERTIFICATE_CREATE_ACME":
		// tos, csr_id and acme_directory_uri default to null in the public
		// model and are then re-read by a per-create_type model that declares
		// them non-nullable, which is the three-error reply the issue reported.
		//
		// dns_mapping defaults to an empty map and survives that second model,
		// then fails a later check that every domain in the CSR has an
		// authenticator ("Please provide DNS authenticator id for ..."). A CSR
		// always has at least one domain, so an empty map can never work and it
		// belongs here with the rest.
		//
		// renew_days is deliberately absent: it defaults to 10 upstream, so
		// requiring it would refuse a configuration that applies cleanly.
		for _, req := range []struct {
			name    string
			summary string
			set     bool
			hint    string
		}{
			{"tos", "Missing tos", !config.TOS.IsNull() && !config.TOS.IsUnknown(),
				"set it to true to accept the ACME provider's terms of service"},
			{"csr_id", "Missing csr_id", !config.CSRID.IsNull() && !config.CSRID.IsUnknown(),
				"point it at an existing CSR, typically another truenas_certificate created with CERTIFICATE_CREATE_CSR"},
			{"acme_directory_uri", "Missing acme_directory_uri", !config.AcmeDirectoryURI.IsNull() && !config.AcmeDirectoryURI.IsUnknown(),
				"for example https://acme-v02.api.letsencrypt.org/directory"},
			{"dns_mapping", "Missing dns_mapping", !config.DNSMapping.IsNull() && !config.DNSMapping.IsUnknown(),
				"map every domain in the CSR to a truenas_acme_dns_authenticator id"},
		} {
			if !req.set {
				resp.Diagnostics.AddAttributeError(
					path.Root(req.name),
					req.summary,
					fmt.Sprintf("create_type=CERTIFICATE_CREATE_ACME requires `%s`: %s.",
						req.name, req.hint),
				)
			}
		}
	}

	// The ACME fields are rejected outright on any other create_type, so say
	// so here rather than letting middleware answer with a field the
	// practitioner did not set on purpose.
	if createType != "CERTIFICATE_CREATE_ACME" {
		for _, f := range []struct {
			name    string
			summary string
			set     bool
		}{
			{"tos", "Invalid tos", !config.TOS.IsNull() && !config.TOS.IsUnknown()},
			{"csr_id", "Invalid csr_id", !config.CSRID.IsNull() && !config.CSRID.IsUnknown()},
			{"acme_directory_uri", "Invalid acme_directory_uri", !config.AcmeDirectoryURI.IsNull() && !config.AcmeDirectoryURI.IsUnknown()},
			{"renew_days", "Invalid renew_days", !config.RenewDays.IsNull() && !config.RenewDays.IsUnknown()},
			{"dns_mapping", "Invalid dns_mapping", !config.DNSMapping.IsNull() && !config.DNSMapping.IsUnknown()},
		} {
			if f.set {
				resp.Diagnostics.AddAttributeError(
					path.Root(f.name),
					f.summary,
					fmt.Sprintf("`%s` applies only to create_type=CERTIFICATE_CREATE_ACME, got %s.",
						f.name, createType),
				)
			}
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

func (r *CertificateResource) mapResponseToModel(ctx context.Context, cert *truenas.Certificate, model *CertificateResourceModel) diag.Diagnostics {
	model.ID = types.StringValue(strconv.Itoa(cert.ID))
	model.Name = types.StringValue(cert.Name)
	model.KeyType = types.StringValue(cert.KeyType)
	model.KeyLength = types.Int64Value(int64(cert.KeyLength))

	// A CSR has no signed certificate behind it, so TrueNAS reports these
	// three as empty for one, whatever was asked for:
	//
	//	"certificate": "", "digest_algorithm": "", "lifetime": 0
	//
	// They are read off the parsed certificate, and there is not one yet. All
	// three are Optional+Computed, so overwriting a configured value with the
	// empty one is "Provider produced inconsistent result after apply", which
	// Terraform calls a provider bug and which is what a CERTIFICATE_CREATE_CSR
	// apply used to end with. Keep what the caller already holds when the API
	// has nothing to say; a create_type that does report these still overwrites
	// normally, so an out-of-band change is still seen.
	model.Certificate = keepKnownString(model.Certificate, cert.CertificateData)
	model.DigestAlgorithm = keepKnownString(model.DigestAlgorithm, cert.DigestAlgorithm)
	model.Lifetime = keepKnownInt64(model.Lifetime, int64(cert.Lifetime))
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

	// SAN from API. The element type has to be the schema's own, not
	// types.StringType: a list built with the wrong element type fails to
	// convert into state, and the error used to be discarded here.
	sanValues := make([]string, 0, len(cert.SAN))
	sanValues = append(sanValues, cert.SAN...)
	san, diags := types.ListValueFrom(ctx, customtypes.SANEntryType{}, sanValues)
	model.SAN = san
	return diags
}

// keepKnownString returns the API's value, or the one already in the model when
// the API returned nothing and the model holds something known.
func keepKnownString(current types.String, apiValue string) types.String {
	if apiValue == "" && !current.IsNull() && !current.IsUnknown() {
		return current
	}
	return types.StringValue(apiValue)
}

// keepKnownInt64 is keepKnownString for a numeric field whose empty is zero.
func keepKnownInt64(current types.Int64, apiValue int64) types.Int64 {
	if apiValue == 0 && !current.IsNull() && !current.IsUnknown() {
		return current
	}
	return types.Int64Value(apiValue)
}
