package types

import "encoding/json"

// ISCSIPortal represents an iSCSI portal.
type ISCSIPortal struct {
	ID      int                 `json:"id"`
	Listen  []ISCSIPortalListen `json:"listen"`
	Comment string              `json:"comment,omitempty"`
	Tag     int                 `json:"tag"`
}

// ISCSIPortalListen represents a portal listen address as returned by the
// read endpoint. SCALE 25.10 no longer accepts `port` in create/update
// requests (the server-side default is 3260 and the field must be omitted),
// but older SCALE versions and read responses may still surface it.
type ISCSIPortalListen struct {
	IP   string `json:"ip"`
	Port int    `json:"port,omitempty"`
}

// iscsiPortalListenWrite is the write-side representation: `port` is
// deliberately absent because SCALE 25.10 rejects unknown fields in
// create/update bodies with HTTP 422 "Extra inputs are not permitted".
type iscsiPortalListenWrite struct {
	IP string `json:"ip"`
}

func toWriteListens(in []ISCSIPortalListen) []iscsiPortalListenWrite {
	out := make([]iscsiPortalListenWrite, 0, len(in))
	for _, l := range in {
		out = append(out, iscsiPortalListenWrite{IP: l.IP})
	}
	return out
}

// ISCSIPortalCreateRequest represents the request to create an iSCSI portal.
type ISCSIPortalCreateRequest struct {
	Listen  []ISCSIPortalListen `json:"-"`
	Comment string              `json:"comment,omitempty"`
}

// MarshalJSON emits a body compatible with SCALE 25.10: only the `ip` key
// is serialized under listen entries; `port` is dropped because the API
// rejects it.
func (r ISCSIPortalCreateRequest) MarshalJSON() ([]byte, error) {
	payload := struct {
		Listen  []iscsiPortalListenWrite `json:"listen"`
		Comment string                   `json:"comment,omitempty"`
	}{
		Listen:  toWriteListens(r.Listen),
		Comment: r.Comment,
	}
	return json.Marshal(payload)
}

// ISCSIPortalUpdateRequest represents the request to update an iSCSI portal.
type ISCSIPortalUpdateRequest struct {
	Listen  []ISCSIPortalListen `json:"-"`
	Comment string              `json:"comment,omitempty"`
}

// MarshalJSON emits a body compatible with SCALE 25.10 (see note on
// ISCSIPortalCreateRequest.MarshalJSON).
func (r ISCSIPortalUpdateRequest) MarshalJSON() ([]byte, error) {
	payload := struct {
		Listen  []iscsiPortalListenWrite `json:"listen,omitempty"`
		Comment string                   `json:"comment,omitempty"`
	}{
		Listen:  toWriteListens(r.Listen),
		Comment: r.Comment,
	}
	return json.Marshal(payload)
}
