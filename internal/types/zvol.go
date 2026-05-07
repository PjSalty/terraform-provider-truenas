package types

// ZvolCreateRequest represents the request to create a zvol.
type ZvolCreateRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Volsize       int64  `json:"volsize"`
	Volblocksize  string `json:"volblocksize,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Compression   string `json:"compression,omitempty"`
	Comments      string `json:"comments,omitempty"`
}

// ZvolUpdateRequest represents the request to update a zvol.
type ZvolUpdateRequest struct {
	Volsize       int64  `json:"volsize,omitempty"`
	Deduplication string `json:"deduplication,omitempty"`
	Compression   string `json:"compression,omitempty"`
	Comments      string `json:"comments,omitempty"`
}
