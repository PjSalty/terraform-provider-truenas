package types

// Catalog represents the TrueNAS SCALE application catalog.
//
// In TrueNAS SCALE 25.04+ the catalog is a singleton, there is only
// one official catalog (label "TRUENAS") and the only user-tunable
// field is preferred_trains. The provider models the full struct for
// state, but only PreferredTrains is mutable through the API.
type Catalog struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	PreferredTrains []string `json:"preferred_trains"`
	Location        string   `json:"location"`
}

// CatalogUpdateRequest is the body for PUT /catalog / catalog.update.
type CatalogUpdateRequest struct {
	PreferredTrains *[]string `json:"preferred_trains,omitempty"`
}
