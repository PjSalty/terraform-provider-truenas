package types

// KerberosRealm represents a Kerberos realm on TrueNAS.
type KerberosRealm struct {
	ID            int      `json:"id"`
	Realm         string   `json:"realm"`
	PrimaryKDC    *string  `json:"primary_kdc,omitempty"`
	KDC           []string `json:"kdc"`
	AdminServer   []string `json:"admin_server"`
	KPasswdServer []string `json:"kpasswd_server"`
}

// KerberosRealmCreateRequest is the body for POST /kerberos/realm /
// kerberos.realm.create.
type KerberosRealmCreateRequest struct {
	Realm         string   `json:"realm"`
	PrimaryKDC    *string  `json:"primary_kdc,omitempty"`
	KDC           []string `json:"kdc"`
	AdminServer   []string `json:"admin_server"`
	KPasswdServer []string `json:"kpasswd_server"`
}

// KerberosRealmUpdateRequest is the body for PUT
// /kerberos/realm/id/{id} / kerberos.realm.update.
type KerberosRealmUpdateRequest struct {
	Realm         *string   `json:"realm,omitempty"`
	PrimaryKDC    *string   `json:"primary_kdc,omitempty"`
	KDC           *[]string `json:"kdc,omitempty"`
	AdminServer   *[]string `json:"admin_server,omitempty"`
	KPasswdServer *[]string `json:"kpasswd_server,omitempty"`
}

// KerberosKeytab represents a Kerberos keytab entry.
type KerberosKeytab struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	File string `json:"file"` // base64-encoded keytab bytes
}

// KerberosKeytabCreateRequest is the body for POST /kerberos/keytab /
// kerberos.keytab.create.
type KerberosKeytabCreateRequest struct {
	Name string `json:"name"`
	File string `json:"file"`
}

// KerberosKeytabUpdateRequest is the body for PUT
// /kerberos/keytab/id/{id} / kerberos.keytab.update.
type KerberosKeytabUpdateRequest struct {
	Name *string `json:"name,omitempty"`
	File *string `json:"file,omitempty"`
}
