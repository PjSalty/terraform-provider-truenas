package types

// ScrubTask represents a ZFS pool scrub task.
type ScrubTask struct {
	ID          int      `json:"id"`
	Pool        int      `json:"pool"`
	PoolName    string   `json:"pool_name"`
	Threshold   int      `json:"threshold"`
	Description string   `json:"description"`
	Schedule    Schedule `json:"schedule"`
	Enabled     bool     `json:"enabled"`
}

// ScrubTaskCreateRequest represents the request to create a scrub task.
type ScrubTaskCreateRequest struct {
	Pool        int      `json:"pool"`
	Threshold   int      `json:"threshold"`
	Description string   `json:"description,omitempty"`
	Schedule    Schedule `json:"schedule"`
	Enabled     bool     `json:"enabled"`
}

// ScrubTaskUpdateRequest represents the request to update a scrub task.
type ScrubTaskUpdateRequest struct {
	Pool        int       `json:"pool,omitempty"`
	Threshold   int       `json:"threshold,omitempty"`
	Description string    `json:"description,omitempty"`
	Schedule    *Schedule `json:"schedule,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
}
