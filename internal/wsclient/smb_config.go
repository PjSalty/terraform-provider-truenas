package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for SMB service: smb.{config,update}.
//
// TrueNAS 26.0 replaced smb.config's boolean enable_smb1 with the tri-state
// minimum_protocol (api/v26_0_0/smb.py:152,175). The models on both sides are
// ConfigDict(extra="forbid", strict=True), so the wrong key is a hard
// ValidationError rather than an ignored field, in BOTH directions:
// minimum_protocol on 25.10 fails exactly as hard as enable_smb1 on 26.
//
// Middleware does ship from_previous/to_previous adapters for this
// (api/v26_0_0/smb.py:298,303), but they only run for a client pinned to an
// older version endpoint. main.py binds "current" straight to the newest
// models with no adapter chain, and the client hardcodes /api/current, so the
// provider gets the raw v26 model and must translate for itself.
//
// The true<->SMB1 / false<->SMB2 mapping is not a guess. It is fixed by
// middleware's own data migration (alembic 26.0 smb-minimum-protocol:
// CASE WHEN cifs_srv_enable_smb1 = 1 THEN 'SMB1' ELSE 'SMB2' END) and by the
// v26 adapter. SMB3 has no legacy representation at all.

// smbUpdateBody is the actual smb.update wire body. It lives here rather than
// in types because it is transport detail: it carries both protocol keys, and
// exactly one of them is ever non-nil.
type smbUpdateBody struct {
	NetbiosName    *string `json:"netbiosname,omitempty"`
	Workgroup      *string `json:"workgroup,omitempty"`
	Description    *string `json:"description,omitempty"`
	UnixCharset    *string `json:"unixcharset,omitempty"`
	AAPLExtensions *bool   `json:"aapl_extensions,omitempty"`
	Guest          *string `json:"guest,omitempty"`
	Filemask       *string `json:"filemask,omitempty"`
	Dirmask        *string `json:"dirmask,omitempty"`

	EnableSMB1      *bool   `json:"enable_smb1,omitempty"`
	MinimumProtocol *string `json:"minimum_protocol,omitempty"`
}

// normalizeSMBConfig fills the derived protocol fields from whichever key the
// server actually sent, and records the dialect.
//
// An unrecognized shape is an error, never a default. Guessing SMB2 here would
// report a fact the server never stated, and on a box that really did have
// SMB1 enabled that guess would silently disable it.
func normalizeSMBConfig(cfg *types.SMBConfig) error {
	switch {
	case cfg.MinimumProtocol != nil:
		cfg.Dialect = types.SMBDialectMinimumProtocol
		cfg.Protocol = *cfg.MinimumProtocol
	case cfg.EnableSMB1 != nil:
		cfg.Dialect = types.SMBDialectEnableSMB1
		if *cfg.EnableSMB1 {
			cfg.Protocol = types.SMBProtocolSMB1
		} else {
			cfg.Protocol = types.SMBProtocolSMB2
		}
	default:
		return fmt.Errorf("smb.config returned neither %q (TrueNAS 25.10 and older) nor %q "+
			"(TrueNAS 26.0 and newer); cannot determine the SMB protocol setting",
			"enable_smb1", "minimum_protocol")
	}
	cfg.SMB1Enabled = cfg.Protocol == types.SMBProtocolSMB1
	return nil
}

// smbDialect reports which protocol key this server speaks, caching the
// answer. GetSMBConfig refreshes the cache on every successful read, so the
// ordinary Read-then-Update sequence costs no extra round trip; only an
// Update against a cold cache issues the probe.
//
// A probe failure is returned, never folded into a guessed dialect.
func (c *Client) smbDialect(ctx context.Context) (types.SMBDialect, error) {
	c.smbDialectMu.Lock()
	cached := c.smbDialectCache
	c.smbDialectMu.Unlock()
	if cached != types.SMBDialectUnknown {
		return cached, nil
	}

	cfg, err := c.GetSMBConfig(ctx)
	if err != nil {
		return types.SMBDialectUnknown, err
	}
	return cfg.Dialect, nil
}

func (c *Client) setSMBDialect(d types.SMBDialect) {
	c.smbDialectMu.Lock()
	c.smbDialectCache = d
	c.smbDialectMu.Unlock()
}

// buildSMBUpdateBody translates a version-agnostic update request into the
// protocol key this server understands.
func buildSMBUpdateBody(req *types.SMBConfigUpdateRequest, d types.SMBDialect) (*smbUpdateBody, error) {
	body := &smbUpdateBody{
		NetbiosName:    req.NetbiosName,
		Workgroup:      req.Workgroup,
		Description:    req.Description,
		UnixCharset:    req.UnixCharset,
		AAPLExtensions: req.AAPLExtensions,
		Guest:          req.Guest,
		Filemask:       req.Filemask,
		Dirmask:        req.Dirmask,
	}

	// No protocol requested means no protocol key on the wire. Emitting one
	// unconditionally is what made the old reset path send enable_smb1 even
	// for users who never set it.
	if req.MinimumProtocol == nil {
		return body, nil
	}

	switch d {
	case types.SMBDialectMinimumProtocol:
		v := *req.MinimumProtocol
		body.MinimumProtocol = &v
	case types.SMBDialectEnableSMB1:
		switch *req.MinimumProtocol {
		case types.SMBProtocolSMB1:
			t := true
			body.EnableSMB1 = &t
		case types.SMBProtocolSMB2:
			f := false
			body.EnableSMB1 = &f
		default:
			// SMB3 cannot be expressed as a boolean. Degrading it to
			// false would quietly apply SMB2 and report success.
			return nil, fmt.Errorf("minimum_protocol %q requires TrueNAS 26.0 or newer; "+
				"this server only has the legacy enable_smb1 boolean, which can express %q and %q",
				*req.MinimumProtocol, types.SMBProtocolSMB1, types.SMBProtocolSMB2)
		}
	default:
		return nil, fmt.Errorf("SMB protocol dialect was not determined; refusing to guess which of "+
			"enable_smb1 or minimum_protocol this server accepts (wanted %q)", *req.MinimumProtocol)
	}
	return body, nil
}

// GetSMBConfig retrieves the SMB service configuration.
func (c *Client) GetSMBConfig(ctx context.Context) (*types.SMBConfig, error) {
	tflog.Trace(ctx, "GetSMBConfig (ws) start")

	result, err := c.Call(ctx, "smb.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting SMB config: %w", err)
	}

	var config types.SMBConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config response: %w", err)
	}
	if err := normalizeSMBConfig(&config); err != nil {
		return nil, fmt.Errorf("getting SMB config: %w", err)
	}
	c.setSMBDialect(config.Dialect)

	tflog.Trace(ctx, "GetSMBConfig (ws) success")
	return &config, nil
}

// UpdateSMBConfig updates the SMB service configuration.
func (c *Client) UpdateSMBConfig(ctx context.Context, req *types.SMBConfigUpdateRequest) (*types.SMBConfig, error) {
	tflog.Trace(ctx, "UpdateSMBConfig (ws) start")

	dialect, err := c.smbDialect(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}
	body, err := buildSMBUpdateBody(req, dialect)
	if err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}

	result, err := c.Call(ctx, "smb.update",
		[]interface{}{body}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}

	var config types.SMBConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config update response: %w", err)
	}
	// smb.update returns the full entry and the resource maps that straight
	// into state, so the write path has to normalize too. Fixing only the
	// read path leaves Create and Update writing a zero-valued protocol.
	if err := normalizeSMBConfig(&config); err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}
	c.setSMBDialect(config.Dialect)

	tflog.Trace(ctx, "UpdateSMBConfig (ws) success")
	return &config, nil
}
