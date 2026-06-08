package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Replication API ---

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

// GetReplication retrieves a replication task by ID.
func (c *Client) GetReplication(ctx context.Context, id int) (*Replication, error) {
	tflog.Trace(ctx, "GetReplication start")

	resp, err := c.Get(ctx, fmt.Sprintf("/replication/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting replication %d: %w", id, err)
	}

	var repl Replication
	if err := json.Unmarshal(resp, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication response: %w", err)
	}

	tflog.Trace(ctx, "GetReplication success")
	return &repl, nil
}

// CreateReplication creates a new replication task.
func (c *Client) CreateReplication(ctx context.Context, req *ReplicationCreateRequest) (*Replication, error) {
	tflog.Trace(ctx, "CreateReplication start")

	resp, err := c.Post(ctx, "/replication", req)
	if err != nil {
		return nil, fmt.Errorf("creating replication: %w", err)
	}

	var repl Replication
	if err := json.Unmarshal(resp, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication create response: %w", err)
	}

	tflog.Trace(ctx, "CreateReplication success")
	return &repl, nil
}

// UpdateReplication updates an existing replication task.
func (c *Client) UpdateReplication(ctx context.Context, id int, req *ReplicationUpdateRequest) (*Replication, error) {
	tflog.Trace(ctx, "UpdateReplication start")

	resp, err := c.Put(ctx, fmt.Sprintf("/replication/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating replication %d: %w", id, err)
	}

	var repl Replication
	if err := json.Unmarshal(resp, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateReplication success")
	return &repl, nil
}

// DeleteReplication deletes a replication task.
func (c *Client) DeleteReplication(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteReplication start")

	_, err := c.Delete(ctx, fmt.Sprintf("/replication/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting replication %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteReplication success")
	return nil
}
