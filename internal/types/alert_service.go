package types

// AlertService represents an alert service configuration returned by
// alertservice.query / alertservice.get_instance / alertservice.create /
// alertservice.update.
//
// SCALE version notes:
//   - 25.04 (legacy): top-level `type` field in both request and response.
//   - 25.10+ (current): `type` lives inside `attributes` as the polymorphic
//     discriminator (per the `oneOf` schema). The endpoint rejects a
//     top-level `type` with "Extra inputs are not permitted".
//
// We model the struct for 25.10+ and require callers to put `type` in
// Settings. GetType() reads it back from either location.
type AlertService struct {
	ID       int                    `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type,omitempty"`
	Enabled  bool                   `json:"enabled"`
	Level    string                 `json:"level"`
	Settings map[string]interface{} `json:"attributes"`
}

// GetType returns the alert service type, preferring the attributes
// embedded form (SCALE 25.10+) and falling back to the 25.04 top-level
// field.
func (a *AlertService) GetType() string {
	if a.Settings != nil {
		if t, ok := a.Settings["type"].(string); ok && t != "" {
			return t
		}
	}
	if a.Type != "" {
		return a.Type
	}
	return ""
}

// AlertServiceCreateRequest represents the request to create an alert
// service. On SCALE 25.10+ `type` MUST be embedded inside attributes
// (the discriminator field of the `oneOf` schema); top-level `type` is
// rejected. Callers should not set the top-level Type field, it
// exists only as a fallback for any pre-25.04 backends.
type AlertServiceCreateRequest struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Level    string                 `json:"level"`
	Settings map[string]interface{} `json:"attributes"`
}

// AlertServiceUpdateRequest represents the request to update an alert service.
type AlertServiceUpdateRequest struct {
	Name     string                 `json:"name,omitempty"`
	Enabled  *bool                  `json:"enabled,omitempty"`
	Level    string                 `json:"level,omitempty"`
	Settings map[string]interface{} `json:"attributes,omitempty"`
}
