package types

import "encoding/json"

// Pool represents a ZFS pool on a TrueNAS system.
type Pool struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	GUID        string `json:"guid"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	Healthy     bool   `json:"healthy"`
	IsDecrypted bool   `json:"is_decrypted"`
}

// PoolCreateRequest is the body for POST /pool / pool.create.
//
// Topology is a raw JSON object so callers can describe the arbitrarily
// nested vdev structure (data/cache/log/spares/special/dedup, each
// containing vdev entries with type + disks + optional draid params)
// without forcing the type package to model every discriminated union
// in the TrueNAS OpenAPI schema.
type PoolCreateRequest struct {
	Name                  string                 `json:"name"`
	Encryption            bool                   `json:"encryption,omitempty"`
	EncryptionOptions     map[string]interface{} `json:"encryption_options,omitempty"`
	Topology              json.RawMessage        `json:"topology"`
	Deduplication         string                 `json:"deduplication,omitempty"`
	Checksum              string                 `json:"checksum,omitempty"`
	AllowDuplicateSerials bool                   `json:"allow_duplicate_serials,omitempty"`
}

// PoolExportRequest is the body for POST /pool/id/{id}/export /
// pool.export. The provider's "delete" path uses Destroy=true to
// actually wipe the pool; Destroy=false leaves the pool intact and
// merely detaches it.
type PoolExportRequest struct {
	Cascade         bool `json:"cascade"`
	RestartServices bool `json:"restart_services"`
	Destroy         bool `json:"destroy"`
}
