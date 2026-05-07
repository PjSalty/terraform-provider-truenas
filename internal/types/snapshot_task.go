package types

// SnapshotTask represents a periodic snapshot task.
type SnapshotTask struct {
	ID           int      `json:"id"`
	Dataset      string   `json:"dataset"`
	Recursive    bool     `json:"recursive"`
	Lifetime     int      `json:"lifetime_value"`
	LifetimeUnit string   `json:"lifetime_unit"`
	NamingSchema string   `json:"naming_schema"`
	Schedule     Schedule `json:"schedule"`
	Enabled      bool     `json:"enabled"`
	AllowEmpty   bool     `json:"allow_empty"`
	Exclude      []string `json:"exclude,omitempty"`
}

// SnapshotTaskCreateRequest represents the request to create a snapshot task.
type SnapshotTaskCreateRequest struct {
	Dataset      string   `json:"dataset"`
	Recursive    bool     `json:"recursive"`
	Lifetime     int      `json:"lifetime_value"`
	LifetimeUnit string   `json:"lifetime_unit"`
	NamingSchema string   `json:"naming_schema"`
	Schedule     Schedule `json:"schedule"`
	Enabled      bool     `json:"enabled"`
	AllowEmpty   bool     `json:"allow_empty"`
	Exclude      []string `json:"exclude,omitempty"`
}

// SnapshotTaskUpdateRequest represents the request to update a snapshot task.
type SnapshotTaskUpdateRequest struct {
	Dataset      string    `json:"dataset,omitempty"`
	Recursive    *bool     `json:"recursive,omitempty"`
	Lifetime     int       `json:"lifetime_value,omitempty"`
	LifetimeUnit string    `json:"lifetime_unit,omitempty"`
	NamingSchema string    `json:"naming_schema,omitempty"`
	Schedule     *Schedule `json:"schedule,omitempty"`
	Enabled      *bool     `json:"enabled,omitempty"`
	AllowEmpty   *bool     `json:"allow_empty,omitempty"`
	Exclude      []string  `json:"exclude,omitempty"`
}
