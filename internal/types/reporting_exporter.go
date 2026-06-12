package types

import "encoding/json"

// ReportingExporter represents a reporting exporter (e.g. GRAPHITE).
// Attributes are polymorphic (discriminated by exporter_type) so we
// keep them as raw JSON.
type ReportingExporter struct {
	ID         int             `json:"id"`
	Enabled    bool            `json:"enabled"`
	Name       string          `json:"name"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// ReportingExporterCreateRequest is the create payload.
type ReportingExporterCreateRequest struct {
	Enabled    bool            `json:"enabled"`
	Name       string          `json:"name"`
	Attributes json.RawMessage `json:"attributes"`
}

// ReportingExporterUpdateRequest is the update payload.
type ReportingExporterUpdateRequest struct {
	Enabled    *bool           `json:"enabled,omitempty"`
	Name       *string         `json:"name,omitempty"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}
