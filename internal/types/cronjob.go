package types

// CronJob represents a cron job in TrueNAS.
type CronJob struct {
	ID          int      `json:"id"`
	User        string   `json:"user"`
	Command     string   `json:"command"`
	Description string   `json:"description,omitempty"`
	Enabled     bool     `json:"enabled"`
	Stdout      bool     `json:"stdout"`
	Stderr      bool     `json:"stderr"`
	Schedule    Schedule `json:"schedule"`
}

// CronJobCreateRequest represents the request to create a cron job.
type CronJobCreateRequest struct {
	User        string   `json:"user"`
	Command     string   `json:"command"`
	Description string   `json:"description,omitempty"`
	Enabled     bool     `json:"enabled"`
	Stdout      bool     `json:"stdout"`
	Stderr      bool     `json:"stderr"`
	Schedule    Schedule `json:"schedule"`
}

// CronJobUpdateRequest represents the request to update a cron job.
type CronJobUpdateRequest struct {
	User        string    `json:"user,omitempty"`
	Command     string    `json:"command,omitempty"`
	Description string    `json:"description,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
	Stdout      *bool     `json:"stdout,omitempty"`
	Stderr      *bool     `json:"stderr,omitempty"`
	Schedule    *Schedule `json:"schedule,omitempty"`
}
