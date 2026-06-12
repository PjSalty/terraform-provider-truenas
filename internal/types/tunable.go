package types

// Tunable is the shape returned by tunable.query / tunable.get_instance
// and accepted by tunable.create / tunable.update.
type Tunable struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Var     string `json:"var"`
	Value   string `json:"value"`
	Comment string `json:"comment"`
	Enabled bool   `json:"enabled"`
}

// TunableCreateRequest is the params object for tunable.create.
type TunableCreateRequest struct {
	Type    string `json:"type"`
	Var     string `json:"var"`
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
	Enabled bool   `json:"enabled"`
}

// TunableUpdateRequest is the params object for tunable.update. All
// fields are optional; only the supplied ones are applied. Enabled is
// a *bool so the difference between "set to false" and "unset" is
// explicit on the wire.
type TunableUpdateRequest struct {
	Value   string `json:"value,omitempty"`
	Comment string `json:"comment,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}
