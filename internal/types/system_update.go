package types

import "encoding/json"

// UpdateTrains is the response shape for "update.get_trains", the map of
// available release trains plus the currently booted train and the train
// selected for update tracking. The `selected` field is what the provider
// reconciles against when the user sets `train` on truenas_system_update;
// `current` is what the box is actually booted on and is surfaced as a
// read-only computed attribute for drift visibility.
type UpdateTrains struct {
	Trains   map[string]UpdateTrainInfo `json:"trains"`
	Current  string                     `json:"current"`
	Selected string                     `json:"selected"`
}

// UpdateTrainInfo is the per-train metadata returned inside UpdateTrains.Trains.
type UpdateTrainInfo struct {
	Description string `json:"description"`
}

// UpdateCheckResult is the response from "update.check_available".
// Status values documented in the TrueNAS API spec:
//   - AVAILABLE: an update is available
//   - UNAVAILABLE: no update available
//   - REBOOT_REQUIRED: an update has already been applied, waiting for cycle
//   - HA_UNAVAILABLE: HA is non-functional
//
// Changes is left as raw JSON because the exact shape is non-stable across
// TrueNAS releases and the resource does not expose it to users, it is
// only surfaced indirectly via the computed `available_version` field.
type UpdateCheckResult struct {
	Status  string          `json:"status"`
	Version string          `json:"version,omitempty"`
	Changes json.RawMessage `json:"changes,omitempty"`
	Notes   string          `json:"notes,omitempty"`
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
