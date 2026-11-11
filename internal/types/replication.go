package types

import (
	"encoding/json"
	"fmt"
)

// TimeSchedule is a cron schedule with a daily time window: the shape of
// replication.schedule and replication.restrict_schedule. TrueNAS has a
// default for every field, so an empty one is left out of a request.
type TimeSchedule struct {
	Minute string `json:"minute,omitempty"`
	Hour   string `json:"hour,omitempty"`
	Dom    string `json:"dom,omitempty"`
	Month  string `json:"month,omitempty"`
	Dow    string `json:"dow,omitempty"`
	Begin  string `json:"begin,omitempty"`
	End    string `json:"end,omitempty"`
}

// Nullable is a request field that TrueNAS takes as a value or as null. It
// has three states, because TrueNAS reads a missing key and a null
// differently: the zero value leaves the key out and TrueNAS keeps what it
// has, Null sends null, and Value sends the value.
type Nullable[T any] struct {
	value *T
	set   bool
}

// Null is a field that is sent as null, which clears it.
func Null[T any]() Nullable[T] {
	return Nullable[T]{set: true}
}

// Value is a field that is sent with v.
func Value[T any](v T) Nullable[T] {
	return Nullable[T]{value: &v, set: true}
}

// IsZero reports whether the field is left out of the request (omitzero).
func (n Nullable[T]) IsZero() bool {
	return !n.set
}

// MarshalJSON writes the value, or null when the field is being cleared.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.value)
}

// ReplicationLifetime is one entry of replication.lifetimes: a retention
// period for the snapshots that match its schedule.
type ReplicationLifetime struct {
	Schedule      Schedule `json:"schedule"`
	LifetimeValue int      `json:"lifetime_value"`
	LifetimeUnit  string   `json:"lifetime_unit"`
}

// ReplicationOptions are the optional fields that a create and an update
// request share. A nil pointer, an empty string or a zero Nullable leaves
// the key out, and TrueNAS then keeps its value or applies its default. A
// pointer to an empty list is sent, and so is an empty map that is not
// nil: both clear the field.
type ReplicationOptions struct {
	Readonly           string                 `json:"readonly,omitempty"`
	AllowFromScratch   *bool                  `json:"allow_from_scratch,omitempty"`
	Retries            *int                   `json:"retries,omitempty"`
	Exclude            *[]string              `json:"exclude,omitempty"`
	Properties         *bool                  `json:"properties,omitempty"`
	PropertiesExclude  *[]string              `json:"properties_exclude,omitempty"`
	PropertiesOverride map[string]string      `json:"properties_override,omitzero"`
	Replicate          *bool                  `json:"replicate,omitempty"`
	Compression        Nullable[string]       `json:"compression,omitzero"`
	SpeedLimit         Nullable[int]          `json:"speed_limit,omitzero"`
	LargeBlock         *bool                  `json:"large_block,omitempty"`
	Embed              *bool                  `json:"embed,omitempty"`
	Compressed         *bool                  `json:"compressed,omitempty"`
	Sudo               *bool                  `json:"sudo,omitempty"`
	LoggingLevel       Nullable[string]       `json:"logging_level,omitzero"`
	NameRegex          Nullable[string]       `json:"name_regex,omitzero"`
	Lifetimes          *[]ReplicationLifetime `json:"lifetimes,omitempty"`

	Encryption                      *bool            `json:"encryption,omitempty"`
	EncryptionInherit               *bool            `json:"encryption_inherit,omitempty"`
	EncryptionKey                   Nullable[string] `json:"encryption_key,omitzero"`
	EncryptionKeyFormat             Nullable[string] `json:"encryption_key_format,omitzero"`
	EncryptionKeyLocation           Nullable[string] `json:"encryption_key_location,omitzero"`
	NetcatActiveSide                Nullable[string] `json:"netcat_active_side,omitzero"`
	NetcatActiveSideListenAddress   Nullable[string] `json:"netcat_active_side_listen_address,omitzero"`
	NetcatActiveSidePortMin         Nullable[int]    `json:"netcat_active_side_port_min,omitzero"`
	NetcatActiveSidePortMax         Nullable[int]    `json:"netcat_active_side_port_max,omitzero"`
	NetcatPassiveSideConnectAddress Nullable[string] `json:"netcat_passive_side_connect_address,omitzero"`
}

// Replication represents a ZFS replication task.
type Replication struct {
	ID                      int           `json:"id"`
	Name                    string        `json:"name"`
	Direction               string        `json:"direction"`
	Transport               string        `json:"transport"`
	SourceDatasets          []string      `json:"source_datasets"`
	TargetDataset           string        `json:"target_dataset"`
	Recursive               bool          `json:"recursive"`
	AutoBool                bool          `json:"auto"`
	Enabled                 bool          `json:"enabled"`
	RetentionPolicy         string        `json:"retention_policy"`
	LifetimeValue           int           `json:"lifetime_value,omitempty"`
	LifetimeUnit            string        `json:"lifetime_unit,omitempty"`
	Schedule                *TimeSchedule `json:"schedule,omitempty"`
	RestrictSchedule        *TimeSchedule `json:"restrict_schedule,omitempty"`
	OnlyMatchingSchedule    bool          `json:"only_matching_schedule"`
	HoldPendingSnapshots    bool          `json:"hold_pending_snapshots"`
	SSHCredentials          int           `json:"ssh_credentials,omitempty"`
	PeriodicSnapshotTasks   []int         `json:"periodic_snapshot_tasks"`
	NamingSchema            []string      `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema []string      `json:"also_include_naming_schema,omitempty"`

	Readonly           string                `json:"readonly"`
	AllowFromScratch   bool                  `json:"allow_from_scratch"`
	Retries            int                   `json:"retries"`
	Exclude            []string              `json:"exclude"`
	Properties         bool                  `json:"properties"`
	PropertiesExclude  []string              `json:"properties_exclude"`
	PropertiesOverride map[string]string     `json:"properties_override"`
	Replicate          bool                  `json:"replicate"`
	Compression        *string               `json:"compression"`
	SpeedLimit         *int                  `json:"speed_limit"`
	LargeBlock         bool                  `json:"large_block"`
	Embed              bool                  `json:"embed"`
	Compressed         bool                  `json:"compressed"`
	Sudo               bool                  `json:"sudo"`
	LoggingLevel       *string               `json:"logging_level"`
	NameRegex          *string               `json:"name_regex"`
	Lifetimes          []ReplicationLifetime `json:"lifetimes"`

	Encryption                      bool    `json:"encryption"`
	EncryptionInherit               *bool   `json:"encryption_inherit"`
	EncryptionKey                   *string `json:"encryption_key"`
	EncryptionKeyFormat             *string `json:"encryption_key_format"`
	EncryptionKeyLocation           *string `json:"encryption_key_location"`
	NetcatActiveSide                *string `json:"netcat_active_side"`
	NetcatActiveSideListenAddress   *string `json:"netcat_active_side_listen_address"`
	NetcatActiveSidePortMin         *int    `json:"netcat_active_side_port_min"`
	NetcatActiveSidePortMax         *int    `json:"netcat_active_side_port_max"`
	NetcatPassiveSideConnectAddress *string `json:"netcat_passive_side_connect_address"`
}

// UnmarshalJSON accepts ssh_credentials and periodic_snapshot_tasks as
// either IDs or expanded objects. Create and update take the IDs, but
// ReplicationEntry, which every response carries, returns the whole
// KeychainCredentialEntry and the whole snapshot task entries, so decoding
// them as integers failed on every SSH replication and would fail on every
// bound snapshot task. The fields always hold IDs; null ssh_credentials
// (LOCAL) leaves it 0. Same API shape, and same handling, as
// CloudSync.credentials.
func (r *Replication) UnmarshalJSON(data []byte) error {
	type Alias Replication
	aux := &struct {
		SSHCredentials        json.RawMessage   `json:"ssh_credentials"`
		PeriodicSnapshotTasks []json.RawMessage `json:"periodic_snapshot_tasks"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	r.PeriodicSnapshotTasks = make([]int, 0, len(aux.PeriodicSnapshotTasks))
	for _, raw := range aux.PeriodicSnapshotTasks {
		id, err := idOrEntryID(raw)
		if err != nil {
			return fmt.Errorf("periodic_snapshot_tasks field: %w", err)
		}
		r.PeriodicSnapshotTasks = append(r.PeriodicSnapshotTasks, id)
	}
	if len(aux.SSHCredentials) == 0 {
		return nil
	}
	id, err := idOrEntryID(aux.SSHCredentials)
	if err != nil {
		return fmt.Errorf("ssh_credentials field: %w", err)
	}
	r.SSHCredentials = id
	return nil
}

// idOrEntryID reads a reference that TrueNAS returns either as a bare ID or
// as the expanded entry with an id key. A null gives 0.
func idOrEntryID(raw json.RawMessage) (int, error) {
	var id int
	if err := json.Unmarshal(raw, &id); err == nil {
		return id, nil
	}
	var entry struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(raw, &entry); err != nil {
		return 0, err
	}
	return entry.ID, nil
}

// ReplicationCreateRequest represents the request to create a replication task.
type ReplicationCreateRequest struct {
	Name                    string                 `json:"name"`
	Direction               string                 `json:"direction"`
	Transport               string                 `json:"transport"`
	SourceDatasets          []string               `json:"source_datasets"`
	TargetDataset           string                 `json:"target_dataset"`
	Recursive               bool                   `json:"recursive"`
	AutoBool                bool                   `json:"auto"`
	Enabled                 bool                   `json:"enabled"`
	RetentionPolicy         string                 `json:"retention_policy"`
	LifetimeValue           int                    `json:"lifetime_value,omitempty"`
	LifetimeUnit            string                 `json:"lifetime_unit,omitempty"`
	Schedule                Nullable[TimeSchedule] `json:"schedule,omitzero"`
	RestrictSchedule        Nullable[TimeSchedule] `json:"restrict_schedule,omitzero"`
	OnlyMatchingSchedule    *bool                  `json:"only_matching_schedule,omitempty"`
	HoldPendingSnapshots    *bool                  `json:"hold_pending_snapshots,omitempty"`
	SSHCredentials          int                    `json:"ssh_credentials,omitempty"`
	PeriodicSnapshotTasks   *[]int                 `json:"periodic_snapshot_tasks,omitempty"`
	NamingSchema            []string               `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema []string               `json:"also_include_naming_schema,omitempty"`
	ReplicationOptions
}

// ReplicationUpdateRequest represents the request to update a replication
// task. A nil pointer leaves the field out, and TrueNAS keeps its value. The
// list pointers exist so an empty list can be sent: it clears the field.
type ReplicationUpdateRequest struct {
	Name                    string                 `json:"name,omitempty"`
	Direction               string                 `json:"direction,omitempty"`
	Transport               string                 `json:"transport,omitempty"`
	SourceDatasets          []string               `json:"source_datasets,omitempty"`
	TargetDataset           string                 `json:"target_dataset,omitempty"`
	Recursive               *bool                  `json:"recursive,omitempty"`
	AutoBool                *bool                  `json:"auto,omitempty"`
	Enabled                 *bool                  `json:"enabled,omitempty"`
	RetentionPolicy         string                 `json:"retention_policy,omitempty"`
	LifetimeValue           int                    `json:"lifetime_value,omitempty"`
	LifetimeUnit            string                 `json:"lifetime_unit,omitempty"`
	Schedule                Nullable[TimeSchedule] `json:"schedule,omitzero"`
	RestrictSchedule        Nullable[TimeSchedule] `json:"restrict_schedule,omitzero"`
	OnlyMatchingSchedule    *bool                  `json:"only_matching_schedule,omitempty"`
	HoldPendingSnapshots    *bool                  `json:"hold_pending_snapshots,omitempty"`
	SSHCredentials          int                    `json:"ssh_credentials,omitempty"`
	PeriodicSnapshotTasks   *[]int                 `json:"periodic_snapshot_tasks,omitempty"`
	NamingSchema            *[]string              `json:"naming_schema,omitempty"`
	AlsoIncludeNamingSchema *[]string              `json:"also_include_naming_schema,omitempty"`
	ReplicationOptions
}
