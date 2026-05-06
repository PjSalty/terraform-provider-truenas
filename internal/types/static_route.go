package types

// StaticRoute represents a static network route in TrueNAS.
type StaticRoute struct {
	ID          int    `json:"id"`
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Description string `json:"description"`
}

// StaticRouteCreateRequest represents the request to create a static route.
type StaticRouteCreateRequest struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Description string `json:"description,omitempty"`
}

// StaticRouteUpdateRequest represents the request to update a static route.
type StaticRouteUpdateRequest struct {
	Destination string `json:"destination,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	Description string `json:"description,omitempty"`
}
