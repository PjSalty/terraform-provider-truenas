package types

// ACMEDNSAuthenticator represents an ACME DNS-01 challenge authenticator
// (Cloudflare, Route53, etc.) registered with the TrueNAS ACME client.
type ACMEDNSAuthenticator struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ACMEDNSAuthenticatorCreateRequest is the body for POST
// /acme/dns/authenticator and JSON-RPC acme.dns.authenticator.create.
// The provider type (e.g. "cloudflare") goes inside Attributes.
type ACMEDNSAuthenticatorCreateRequest struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ACMEDNSAuthenticatorUpdateRequest is the body for PUT
// /acme/dns/authenticator/id/{id} and JSON-RPC
// acme.dns.authenticator.update.
type ACMEDNSAuthenticatorUpdateRequest struct {
	Name       string                 `json:"name,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
