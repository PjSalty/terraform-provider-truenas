package types

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
