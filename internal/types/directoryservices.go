package types

// DirectoryServicesConfig represents the directory services singleton
// config.
//
// The API uses polymorphic credential and configuration objects whose
// shape depends on service_type (ACTIVEDIRECTORY|IPA|LDAP) and
// credential_type. We capture these as raw JSON maps so the resource
// can round-trip them without modeling every variant.
type DirectoryServicesConfig struct {
	ID                 int                    `json:"id"`
	ServiceType        *string                `json:"service_type,omitempty"`
	Credential         map[string]interface{} `json:"credential,omitempty"`
	Enable             bool                   `json:"enable"`
	EnableAccountCache bool                   `json:"enable_account_cache"`
	EnableDNSUpdates   bool                   `json:"enable_dns_updates"`
	Timeout            int                    `json:"timeout"`
	KerberosRealm      *string                `json:"kerberos_realm,omitempty"`
	Configuration      map[string]interface{} `json:"configuration,omitempty"`
}

// DirectoryServicesUpdateRequest is the body for PUT /directoryservices
// / directoryservices.update.
//
// Any field left as nil is omitted, matching the singleton-patch
// pattern used by other config resources.
type DirectoryServicesUpdateRequest struct {
	ServiceType        *string                 `json:"service_type,omitempty"`
	Credential         *map[string]interface{} `json:"credential,omitempty"`
	Enable             *bool                   `json:"enable,omitempty"`
	EnableAccountCache *bool                   `json:"enable_account_cache,omitempty"`
	EnableDNSUpdates   *bool                   `json:"enable_dns_updates,omitempty"`
	Timeout            *int                    `json:"timeout,omitempty"`
	KerberosRealm      *string                 `json:"kerberos_realm,omitempty"`
	Configuration      *map[string]interface{} `json:"configuration,omitempty"`
	Force              *bool                   `json:"force,omitempty"`
}
