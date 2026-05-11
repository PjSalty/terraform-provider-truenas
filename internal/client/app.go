package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- App API ---

// App represents a deployed TrueNAS SCALE application (Docker/iX).
//
// The TrueNAS /app API uses the app name as the string ID.
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

// AppCreateRequest represents the body for POST /app.
//
// `Values` is a free-form object — the provider resource serializes it from a
// user-supplied JSON string so arbitrary Helm/values-style configuration can
// be passed without hard-coding a schema.
type AppCreateRequest struct {
	AppName    string                 `json:"app_name"`
	CatalogApp string                 `json:"catalog_app,omitempty"`
	Train      string                 `json:"train,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Values     map[string]interface{} `json:"values,omitempty"`
	CustomApp  bool                   `json:"custom_app,omitempty"`
}

// AppUpdateRequest represents the body for PUT /app/id/{id_}.
//
// Only `values` (and custom compose fields) are accepted on update; changes to
// train/version/catalog_app require a dedicated upgrade endpoint and are
// modeled as RequiresReplace in the resource schema.
type AppUpdateRequest struct {
	Values map[string]interface{} `json:"values,omitempty"`
}

// AppDeleteRequest represents the body for DELETE /app/id/{id_}.
type AppDeleteRequest struct {
	RemoveImages    bool `json:"remove_images"`
	RemoveIxVolumes bool `json:"remove_ix_volumes"`
}

// ListApps retrieves all deployed apps.
func (c *Client) ListApps(ctx context.Context) ([]App, error) {
	tflog.Trace(ctx, "ListApps start")

	resp, err := c.Get(ctx, "/app")
	if err != nil {
		return nil, fmt.Errorf("listing apps: %w", err)
	}

	var apps []App
	if err := json.Unmarshal(resp, &apps); err != nil {
		return nil, fmt.Errorf("parsing apps list response: %w", err)
	}
	tflog.Trace(ctx, "ListApps success")
	return apps, nil
}

// GetApp retrieves a deployed app by its string ID (app name).
//
// The /app/id/{id_} endpoint returns a single object (not a list).
func (c *Client) GetApp(ctx context.Context, id string) (*App, error) {
	tflog.Trace(ctx, "GetApp start")

	encoded := url.PathEscape(id)
	resp, err := c.Get(ctx, fmt.Sprintf("/app/id/%s", encoded))
	if err != nil {
		return nil, fmt.Errorf("getting app %q: %w", id, err)
	}

	// The API sometimes returns a bare object and sometimes a single-element
	// list depending on call style; handle both.
	var app App
	if err := json.Unmarshal(resp, &app); err == nil && app.ID != "" {
		return &app, nil
	}

	var apps []App
	if err := json.Unmarshal(resp, &apps); err != nil {
		return nil, fmt.Errorf("parsing app response: %w", err)
	}
	if len(apps) == 0 {
		return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("app %q not found", id)}
	}
	tflog.Trace(ctx, "GetApp success")
	return &apps[0], nil
}

// CreateApp installs a new app. POST /app is async and returns a job ID.
func (c *Client) CreateApp(ctx context.Context, req *AppCreateRequest) (*App, error) {
	tflog.Trace(ctx, "CreateApp start")

	resp, err := c.Post(ctx, "/app", req)
	if err != nil {
		return nil, fmt.Errorf("creating app %q: %w", req.AppName, err)
	}

	var jobID int
	if err := json.Unmarshal(resp, &jobID); err != nil {
		return nil, fmt.Errorf("parsing app create job ID: %w", err)
	}

	if _, err := c.WaitForJob(ctx, jobID); err != nil {
		return nil, fmt.Errorf("waiting for app create job: %w", err)
	}

	// The job result may contain the created app, but fetch fresh to avoid
	// relying on that contract and to get an up-to-date state.
	tflog.Trace(ctx, "CreateApp success")
	return c.GetApp(ctx, req.AppName)
}

// UpdateApp updates an existing app. PUT /app/id/{id_} is async.
func (c *Client) UpdateApp(ctx context.Context, id string, req *AppUpdateRequest) (*App, error) {
	tflog.Trace(ctx, "UpdateApp start")

	encoded := url.PathEscape(id)
	resp, err := c.Put(ctx, fmt.Sprintf("/app/id/%s", encoded), req)
	if err != nil {
		return nil, fmt.Errorf("updating app %q: %w", id, err)
	}

	var jobID int
	if err := json.Unmarshal(resp, &jobID); err != nil {
		// Fall back to treating the response as the app itself.
		var app App
		if err2 := json.Unmarshal(resp, &app); err2 == nil && app.ID != "" {
			return &app, nil
		}
		return nil, fmt.Errorf("parsing app update response: %w", err)
	}

	if _, err := c.WaitForJob(ctx, jobID); err != nil {
		return nil, fmt.Errorf("waiting for app update job: %w", err)
	}
	tflog.Trace(ctx, "UpdateApp success")
	return c.GetApp(ctx, id)
}

// DeleteApp deletes an app. DELETE /app/id/{id_} is async and accepts a body.
func (c *Client) DeleteApp(ctx context.Context, id string, req *AppDeleteRequest) error {
	tflog.Trace(ctx, "DeleteApp start")

	encoded := url.PathEscape(id)
	resp, err := c.DeleteWithBody(ctx, fmt.Sprintf("/app/id/%s", encoded), req)
	if err != nil {
		return fmt.Errorf("deleting app %q: %w", id, err)
	}

	if err := c.waitIfJobResponse(ctx, resp, fmt.Sprintf("delete app %q", id)); err != nil {
		return err
	}
	tflog.Trace(ctx, "DeleteApp success")
	return nil
}

// --- Catalog API ---

// Catalog represents the TrueNAS SCALE application catalog.
//
// In TrueNAS SCALE 25.04+ the catalog is a singleton — there is only one
// official catalog (label "TRUENAS") and the only user-tunable field is
// `preferred_trains`. The provider models the full struct for state, but
// only `preferred_trains` is mutable through the REST API.
type Catalog struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	PreferredTrains []string `json:"preferred_trains"`
	Location        string   `json:"location"`
}

// CatalogUpdateRequest represents the body for PUT /catalog.
type CatalogUpdateRequest struct {
	PreferredTrains *[]string `json:"preferred_trains,omitempty"`
}

// GetCatalog retrieves the singleton catalog configuration.
func (c *Client) GetCatalog(ctx context.Context) (*Catalog, error) {
	tflog.Trace(ctx, "GetCatalog start")

	resp, err := c.Get(ctx, "/catalog")
	if err != nil {
		return nil, fmt.Errorf("getting catalog: %w", err)
	}

	var cat Catalog
	if err := json.Unmarshal(resp, &cat); err != nil {
		return nil, fmt.Errorf("parsing catalog response: %w", err)
	}
	tflog.Trace(ctx, "GetCatalog success")
	return &cat, nil
}

// UpdateCatalog updates the singleton catalog configuration.
func (c *Client) UpdateCatalog(ctx context.Context, req *CatalogUpdateRequest) (*Catalog, error) {
	tflog.Trace(ctx, "UpdateCatalog start")

	resp, err := c.Put(ctx, "/catalog", req)
	if err != nil {
		return nil, fmt.Errorf("updating catalog: %w", err)
	}

	var cat Catalog
	if err := json.Unmarshal(resp, &cat); err != nil {
		return nil, fmt.Errorf("parsing catalog update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateCatalog success")
	return &cat, nil
}

// SyncCatalog triggers a catalog sync. This is an async job, but the REST API
// surfaces it as GET /catalog/sync returning a job ID.
func (c *Client) SyncCatalog(ctx context.Context) error {
	tflog.Trace(ctx, "SyncCatalog start")

	resp, err := c.Get(ctx, "/catalog/sync")
	if err != nil {
		return fmt.Errorf("triggering catalog sync: %w", err)
	}

	if err := c.waitIfJobResponse(ctx, resp, "catalog sync"); err != nil {
		return err
	}
	tflog.Trace(ctx, "SyncCatalog success")
	return nil
}
