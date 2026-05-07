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
}

// CertificateUpdateRequest is the body for PUT /certificate/id/{id} /
// JSON-RPC certificate.update. Only Name is mutable post-creation.
type CertificateUpdateRequest struct {
	Name string `json:"name,omitempty"`
}
