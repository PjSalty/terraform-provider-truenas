package types

// LXCConfig is the system-wide LXC container configuration, new in
// TrueNAS 26.0. The lxc namespace does not exist on 25.10 or earlier.
//
// It is a ConfigService singleton: one row, read with lxc.config and
// written with lxc.update.
//
// PreferredPool and Bridge are nullable upstream, so they are pointers:
// "not configured" is a real state, distinct from the empty string.
type LXCConfig struct {
	ID            int     `json:"id"`
	PreferredPool *string `json:"preferred_pool"`
	Bridge        *string `json:"bridge"`
	V4Network     string  `json:"v4_network"`
	V6Network     string  `json:"v6_network"`
}

// LXCConfigUpdateRequest is the body for lxc.update. Every field is
// optional: the upstream model is ForUpdateMetaclass, so an omitted key
// leaves the stored value alone.
type LXCConfigUpdateRequest struct {
	PreferredPool *string `json:"preferred_pool,omitempty"`
	Bridge        *string `json:"bridge,omitempty"`
	V4Network     *string `json:"v4_network,omitempty"`
	V6Network     *string `json:"v6_network,omitempty"`
}
