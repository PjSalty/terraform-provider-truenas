package types

// Certificate represents a TLS certificate in TrueNAS.
//
// Returned by both REST GET /certificate/id/{id} and JSON-RPC
// certificate.query. Values are identical between transports.
type Certificate struct {
	ID                 int      `json:"id"`
	Type               int      `json:"type"`
	Name               string   `json:"name"`
	CertificateData    string   `json:"certificate"`
	Privatekey         string   `json:"privatekey"`
	CSR                string   `json:"CSR"`
	KeyLength          int      `json:"key_length"`
	KeyType            string   `json:"key_type"`
	Country            string   `json:"country"`
	State              string   `json:"state"`
	City               string   `json:"city"`
	Organization       string   `json:"organization"`
	OrganizationalUnit string   `json:"organizational_unit"`
	Common             string   `json:"common"`
	Email              string   `json:"email"`
	DigestAlgorithm    string   `json:"digest_algorithm"`
	Lifetime           int      `json:"lifetime"`
	From               string   `json:"from"`
	Until              string   `json:"until"`
	Expired            bool     `json:"expired"`
	Parsed             bool     `json:"parsed"`
	DN                 string   `json:"DN"`
	SAN                []string `json:"san"`

	// How the certificate came to exist. certificate.query reports this, so
	// ImportState does not have to assume: cert_type_CSR marks a request that
	// has not been signed yet, and acme_uri is set only on one issued through
	// ACME.
	CertTypeCSR bool   `json:"cert_type_CSR"`
	AcmeURI     string `json:"acme_uri"`
}

// CertificateCreateRequest is the body for POST /certificate /
// JSON-RPC certificate.create. Both transports return a job ID; the
// client implementation polls until terminal state.
type CertificateCreateRequest struct {
	Name               string   `json:"name"`
	CreateType         string   `json:"create_type"`
	CertificateData    string   `json:"certificate,omitempty"`
	Privatekey         string   `json:"privatekey,omitempty"`
	KeyType            string   `json:"key_type,omitempty"`
	KeyLength          int      `json:"key_length,omitempty"`
	DigestAlgorithm    string   `json:"digest_algorithm,omitempty"`
	Country            string   `json:"country,omitempty"`
	State              string   `json:"state,omitempty"`
	City               string   `json:"city,omitempty"`
	Organization       string   `json:"organization,omitempty"`
	OrganizationalUnit string   `json:"organizational_unit,omitempty"`
	Email              string   `json:"email,omitempty"`
	Common             string   `json:"common,omitempty"`
	SAN                []string `json:"san,omitempty"`

	// create_type = CERTIFICATE_CREATE_IMPORTED_CSR. That model is
	// name + CSR + privatekey, and the provider had no way to send the CSR
	// itself, so the create_type was offered by the schema and could never
	// succeed. Same shape as the ACME gap.
	CSR string `json:"CSR,omitempty"`

	// ACME, for create_type = CERTIFICATE_CREATE_ACME. Middleware validates
	// that create_type against CertificateCreateACMEArgs, where tos, csr_id,
	// renew_days, acme_directory_uri and dns_mapping are all declared with no
	// default. Omitting them is what made that create_type impossible:
	//
	//	[EINVAL] certificate_create_acme.tos: Input should be a valid boolean
	//	[EINVAL] certificate_create_acme.csr_id: Input should be a valid integer
	//	[EINVAL] certificate_create_acme.acme_directory_uri: Input should be a valid string
	//
	// Tos is a *bool on purpose. A plain bool with omitempty drops
	// `tos = false`, the server sees nothing, and reports the same error the
	// practitioner was trying to fix.
	AcmeDirectoryURI string         `json:"acme_directory_uri,omitempty"`
	CSRID            *int           `json:"csr_id,omitempty"`
	TOS              *bool          `json:"tos,omitempty"`
	RenewDays        *int           `json:"renew_days,omitempty"`
	DNSMapping       map[string]int `json:"dns_mapping,omitempty"`
}

// CertificateUpdateRequest is the body for certificate.update. Upstream's
// CertificateUpdate model carries exactly three fields; everything else about
// a certificate is fixed at creation. add_to_trusted_store is not modeled by
// this provider yet.
//
// RenewDays is a pointer so an unset value is omitted rather than sent as 0,
// which is outside the 1..30 the server accepts.
type CertificateUpdateRequest struct {
	Name      string `json:"name,omitempty"`
	RenewDays *int   `json:"renew_days,omitempty"`
}
