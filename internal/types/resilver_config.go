package types

// ResilverConfig is the pool resilver priority window. Inside the window
// TrueNAS raises the ZFS resilver tunables so a resilver finishes sooner,
// and outside it puts the defaults back.
//
// It is a ConfigService singleton: one row, read with pool.resilver.config
// and written with pool.resilver.update.
//
// Begin and End are "HH:MM" in 24 hour time. A Begin later than End rolls
// over midnight. Weekday holds 1 (Monday) through 7 (Sunday).
type ResilverConfig struct {
	ID      int    `json:"id"`
	Enabled bool   `json:"enabled"`
	Begin   string `json:"begin"`
	End     string `json:"end"`
	Weekday []int  `json:"weekday"`
}

// ResilverConfigUpdateRequest is the body for pool.resilver.update. Every
// field is optional: the upstream model is ForUpdateMetaclass, so an
// omitted key leaves the stored value alone. TrueNAS rejects an empty
// Weekday, so leaving an empty one out loses nothing.
type ResilverConfigUpdateRequest struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Begin   *string `json:"begin,omitempty"`
	End     *string `json:"end,omitempty"`
	Weekday []int   `json:"weekday,omitempty"`
}
