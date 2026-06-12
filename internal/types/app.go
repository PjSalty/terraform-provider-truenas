package types

import "encoding/json"

// App represents a deployed TrueNAS SCALE application (Docker/iX).
//
// The TrueNAS app API uses the app name as the string ID.
type App struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	State            string          `json:"state"`
	UpgradeAvailable bool            `json:"upgrade_available"`
	LatestVersion    string          `json:"latest_version"`
	HumanVersion     string          `json:"human_version"`
	Version          string          `json:"version"`
	CustomApp        bool            `json:"custom_app"`
	Migrated         bool            `json:"migrated"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

// AppCreateRequest represents the body for POST /app / app.create.
//
// Values is a free-form object, the resource serializes it from a
// user-supplied JSON string so arbitrary Helm/values-style configuration
// can pass through without hard-coding a schema.
type AppCreateRequest struct {
	AppName    string                 `json:"app_name"`
	CatalogApp string                 `json:"catalog_app,omitempty"`
	Train      string                 `json:"train,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Values     map[string]interface{} `json:"values,omitempty"`
	CustomApp  bool                   `json:"custom_app,omitempty"`
}

// AppUpdateRequest represents the body for PUT /app/id/{id_} /
// app.update.
//
// Only Values (and custom compose fields) are accepted on update;
// changes to train/version/catalog_app require a dedicated upgrade
// endpoint and are modeled as RequiresReplace in the resource schema.
type AppUpdateRequest struct {
	Values map[string]interface{} `json:"values,omitempty"`
}

// AppDeleteRequest represents the body for DELETE /app/id/{id_} /
// app.delete.
type AppDeleteRequest struct {
	RemoveImages    bool `json:"remove_images"`
	RemoveIxVolumes bool `json:"remove_ix_volumes"`
}
