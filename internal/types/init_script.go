package types

// InitScript represents an init/startup script in TrueNAS, returned by
// initshutdownscript.query / initshutdownscript.get_instance and
// returned by initshutdownscript.create / initshutdownscript.update.
type InitScript struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	Script  string `json:"script,omitempty"`
	When    string `json:"when"`
	Enabled bool   `json:"enabled"`
	Timeout int    `json:"timeout"`
	Comment string `json:"comment,omitempty"`
}

// InitScriptCreateRequest is the params object for initshutdownscript.create.
type InitScriptCreateRequest struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	Script  string `json:"script,omitempty"`
	When    string `json:"when"`
	Enabled bool   `json:"enabled"`
	Timeout int    `json:"timeout,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// InitScriptUpdateRequest is the partial-update params object for
// initshutdownscript.update. All fields are optional; Enabled is a
// *bool so unset is distinguishable from explicit false on the wire.
type InitScriptUpdateRequest struct {
	Type    string `json:"type,omitempty"`
	Command string `json:"command,omitempty"`
	Script  string `json:"script,omitempty"`
	When    string `json:"when,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
	Comment string `json:"comment,omitempty"`
}
