package types

// UpdateConfig is the system update configuration, returned by update.config
// and update.update.
//
// This replaced the train/auto-download surface during the 25.10.0 migration
// of the update service to a config service. Profile is nullable on the
// "safe" entry model, so it is a pointer here; middleware only guarantees a
// value once one has been selected.
type UpdateConfig struct {
	ID        int     `json:"id"`
	Autocheck bool    `json:"autocheck"`
	Profile   *string `json:"profile"`
}

// UpdateConfigUpdateRequest is the body for update.update. Both fields are
// pointers so an unset attribute is omitted rather than sent as a zero value,
// which would silently turn autocheck off.
type UpdateConfigUpdateRequest struct {
	Autocheck *bool   `json:"autocheck,omitempty"`
	Profile   *string `json:"profile,omitempty"`
}

// UpdateProfileChoice is one entry of update.profile_choices.
//
// Available is the one that matters: middleware rejects selecting a profile
// that is not available, so the provider checks it rather than letting the
// apply fail server-side.
type UpdateProfileChoice struct {
	Name        string `json:"name"`
	Footnote    string `json:"footnote"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
}

// UpdateStatus is the response from update.status, which replaced
// update.check_available.
//
// Status and Error are nullable on the wire, hence pointers: a nil Status is
// "no information", not "no update available".
type UpdateStatus struct {
	Code   string              `json:"code"` // NORMAL | ERROR
	Status *UpdateStatusDetail `json:"status"`
	Error  *UpdateStatusError  `json:"error"`
}

// UpdateStatusDetail carries the running and candidate versions.
type UpdateStatusDetail struct {
	CurrentVersion UpdateVersionInfo  `json:"current_version"`
	NewVersion     *UpdateVersionInfo `json:"new_version"`
}

// UpdateVersionInfo is a version plus the profile it belongs to.
type UpdateVersionInfo struct {
	Version string `json:"version"`
	Profile string `json:"profile"`
}

// UpdateStatusError is the error detail when Code is ERROR.
type UpdateStatusError struct {
	Errname string `json:"errname"`
	Reason  string `json:"reason"`
}

// SystemInfo is the shape returned by "system.info". A small subset is
// surfaced directly on resources (truenas_system_update reads Version);
// the rest is held for diagnostic completeness so future resources can
// adopt fields without re-defining the struct.
type SystemInfo struct {
	Version       string  `json:"version"`
	Hostname      string  `json:"hostname"`
	PhysicalMem   int64   `json:"physmem"`
	Model         string  `json:"model"`
	Cores         int     `json:"cores"`
	Uptime        string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	DateTime      struct {
		Year   int    `json:"year"`
		Month  int    `json:"month"`
		Day    int    `json:"day"`
		Hour   int    `json:"hour"`
		Minute int    `json:"minute"`
		Second int    `json:"second"`
		TZ     string `json:"timezone"`
	} `json:"datetime"`
	SystemSerial  string    `json:"system_serial"`
	SystemProduct string    `json:"system_product"`
	Timezone      string    `json:"timezone"`
	Loadavg       []float64 `json:"loadavg"`
}
