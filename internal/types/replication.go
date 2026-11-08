package types

import (
	"encoding/json"
	"fmt"
)

// Replication represents a ZFS replication task.
type Replication struct {
	ID                      int       `json:"id"`
	Name                    string    `json:"name"`
	Direction               string    `json:"direction"`
	Transport               string    `json:"transport"`
	SourceDatasets          []string  `json:"source_datasets"`
	TargetDataset           string    `json:"target_dataset"`
	Recursive               bool      `json:"recursive"`
	AutoBool                bool      `json:"auto"`
	Enabled                 bool      `json:"enabled"`
	RetentionPolicy         string    `json:"retention_policy"`
	LifetimeValue           int       `json:"lifetime_value,omitempty"`
	LifetimeUnit            string    `json:"lifetime_unit,omitempty"`
	Schedule                *Schedule `json:"schedule,omitempty"`
	SSHCredentials          int       `json:"ssh_credentials,omitempty"`
	NamingSchema            []string  `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema []string  `json:"also_include_naming_schema,omitempty"`
}

// UnmarshalJSON accepts ssh_credentials as either the integer ID or the
// expanded credential object. Create and update take the ID, but
// ReplicationEntry, which every response carries, returns the whole
// KeychainCredentialEntry, so decoding it as an int failed on every SSH
// replication. The field always holds the ID; null (LOCAL) leaves it 0.
// Same API shape, and same handling, as CloudSync.credentials.
func (r *Replication) UnmarshalJSON(data []byte) error {
	type Alias Replication
	aux := &struct {
		SSHCredentials json.RawMessage `json:"ssh_credentials"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.SSHCredentials) == 0 {
		return nil
	}
	var id int
	if err := json.Unmarshal(aux.SSHCredentials, &id); err == nil {
		r.SSHCredentials = id
		return nil
	}
	var obj struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(aux.SSHCredentials, &obj); err != nil {
		return fmt.Errorf("ssh_credentials field: %w", err)
	}
	r.SSHCredentials = obj.ID
	return nil
}

// ReplicationCreateRequest represents the request to create a replication task.
type ReplicationCreateRequest struct {
	Name                    string    `json:"name"`
	Direction               string    `json:"direction"`
	Transport               string    `json:"transport"`
	SourceDatasets          []string  `json:"source_datasets"`
	TargetDataset           string    `json:"target_dataset"`
	Recursive               bool      `json:"recursive"`
	AutoBool                bool      `json:"auto"`
	Enabled                 bool      `json:"enabled"`
	RetentionPolicy         string    `json:"retention_policy"`
	LifetimeValue           int       `json:"lifetime_value,omitempty"`
	LifetimeUnit            string    `json:"lifetime_unit,omitempty"`
	Schedule                *Schedule `json:"schedule,omitempty"`
	SSHCredentials          int       `json:"ssh_credentials,omitempty"`
	NamingSchema            []string  `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema []string  `json:"also_include_naming_schema,omitempty"`
}

// ReplicationUpdateRequest represents the request to update a replication task.
type ReplicationUpdateRequest struct {
	Name            string    `json:"name,omitempty"`
	Direction       string    `json:"direction,omitempty"`
	Transport       string    `json:"transport,omitempty"`
	SourceDatasets  []string  `json:"source_datasets,omitempty"`
	TargetDataset   string    `json:"target_dataset,omitempty"`
	Recursive       *bool     `json:"recursive,omitempty"`
	AutoBool        *bool     `json:"auto,omitempty"`
	Enabled         *bool     `json:"enabled,omitempty"`
	RetentionPolicy string    `json:"retention_policy,omitempty"`
	LifetimeValue   int       `json:"lifetime_value,omitempty"`
	LifetimeUnit    string    `json:"lifetime_unit,omitempty"`
	Schedule        *Schedule `json:"schedule,omitempty"`
	SSHCredentials  int       `json:"ssh_credentials,omitempty"`
}
