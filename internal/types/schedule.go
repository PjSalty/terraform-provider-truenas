package types

// Schedule is the cron-style schedule shape used by cronjob, snapshot task,
// scrub task, cloud sync, rsync task, and replication.
type Schedule struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
}
