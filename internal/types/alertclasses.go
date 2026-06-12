package types

// AlertClassEntry is the per-class configuration row in the
// alertclasses singleton. All fields are optional; an empty entry
// inherits the system default.
type AlertClassEntry struct {
	Level            string `json:"level,omitempty"`
	Policy           string `json:"policy,omitempty"`
	ProactiveSupport *bool  `json:"proactive_support,omitempty"`
}

// AlertClassesConfig is the singleton alert classes configuration
// returned by alertclasses.config and alertclasses.update.
type AlertClassesConfig struct {
	ID      int                        `json:"id"`
	Classes map[string]AlertClassEntry `json:"classes"`
}

// AlertClassesUpdateRequest is the params object for alertclasses.update.
type AlertClassesUpdateRequest struct {
	Classes map[string]AlertClassEntry `json:"classes"`
}
