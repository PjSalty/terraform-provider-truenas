package types

// RsyncTask represents an rsync task in TrueNAS.
type RsyncTask struct {
	ID           int      `json:"id"`
	Path         string   `json:"path"`
	Remotehost   string   `json:"remotehost,omitempty"`
	Remoteport   int      `json:"remoteport,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	Remotemodule string   `json:"remotemodule,omitempty"`
	Remotepath   string   `json:"remotepath,omitempty"`
	Direction    string   `json:"direction,omitempty"`
	Schedule     Schedule `json:"schedule"`
	User         string   `json:"user"`
	Enabled      bool     `json:"enabled"`
	Desc         string   `json:"desc,omitempty"`
}

// RsyncTaskCreateRequest represents the request to create an rsync task.
type RsyncTaskCreateRequest struct {
	Path         string   `json:"path"`
	Remotehost   string   `json:"remotehost,omitempty"`
	Remoteport   int      `json:"remoteport,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	Remotemodule string   `json:"remotemodule,omitempty"`
	Remotepath   string   `json:"remotepath,omitempty"`
	Direction    string   `json:"direction,omitempty"`
	Schedule     Schedule `json:"schedule,omitempty"`
	User         string   `json:"user"`
	Enabled      bool     `json:"enabled"`
	Desc         string   `json:"desc,omitempty"`
}

// RsyncTaskUpdateRequest represents the request to update an rsync task.
type RsyncTaskUpdateRequest struct {
	Path         string    `json:"path,omitempty"`
	Remotehost   string    `json:"remotehost,omitempty"`
	Remoteport   int       `json:"remoteport,omitempty"`
	Mode         string    `json:"mode,omitempty"`
	Remotemodule string    `json:"remotemodule,omitempty"`
	Remotepath   string    `json:"remotepath,omitempty"`
	Direction    string    `json:"direction,omitempty"`
	Schedule     *Schedule `json:"schedule,omitempty"`
	User         string    `json:"user,omitempty"`
	Enabled      *bool     `json:"enabled,omitempty"`
	Desc         string    `json:"desc,omitempty"`
}
