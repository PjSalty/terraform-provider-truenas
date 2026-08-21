package types

// Webshare is a WebShare share, new in TrueNAS 26.0. There is no
// counterpart in 25.10 or earlier: api/v26_0_0/webshare.py has no
// v25_10_5 equivalent, and the sharing.webshare namespace does not exist
// on those releases.
//
// Only Name, Path, Enabled and IsHomeBase are writable. Dataset,
// RelativePath, Locked and Tier are excluded_field() on
// SharingWebshareCreate upstream, so they are derived by middleware and
// read-only here. Modeling them as settable would produce a plan the
// server silently ignores.
type Webshare struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Path         string  `json:"path"`
	Enabled      bool    `json:"enabled"`
	IsHomeBase   bool    `json:"is_home_base"`
	Dataset      *string `json:"dataset"`
	RelativePath *string `json:"relative_path"`
	Locked       *bool   `json:"locked"`
}

// WebshareCreateRequest is the body for sharing.webshare.create.
//
// Enabled and IsHomeBase are pointers so an unset attribute is omitted
// rather than sent as false, which would override the server-side
// default of enabled=true.
type WebshareCreateRequest struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Enabled    *bool  `json:"enabled,omitempty"`
	IsHomeBase *bool  `json:"is_home_base,omitempty"`
}

// WebshareUpdateRequest is the body for sharing.webshare.update. Every
// field is optional: the upstream model is ForUpdateMetaclass, so an
// omitted key leaves the stored value alone.
type WebshareUpdateRequest struct {
	Name       *string `json:"name,omitempty"`
	Path       *string `json:"path,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
	IsHomeBase *bool   `json:"is_home_base,omitempty"`
}
