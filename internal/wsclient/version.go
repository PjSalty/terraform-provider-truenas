package wsclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Server version detection.
//
// Several fields changed meaning or disappeared across TrueNAS releases in ways
// no version adapter can rescue, because the client dials /api/current and so
// always gets the newest models (see the note in smb_config.go). Where the
// provider has to behave differently per release, it asks the server what it is
// rather than guessing, and it says so in the diagnostic when it refuses.
//
// system.info is the probe: it is present on every release the provider
// supports, is read-only, and its `version` field is the release string
// ("25.10.4", "26.0.0-BETA.2").

// ServerVersion is a comparable TrueNAS release. Only major and minor are kept:
// every behavioral difference the provider cares about lands on a minor
// boundary, and patch/BETA suffixes vary in shape.
type ServerVersion struct {
	Major int
	Minor int
	Raw   string
}

// String returns the raw version as the server reported it, which is what a
// diagnostic should show the operator.
func (v ServerVersion) String() string { return v.Raw }

// AtLeast reports whether the server is at or above major.minor.
func (v ServerVersion) AtLeast(major, minor int) bool {
	if v.Major != major {
		return v.Major > major
	}
	return v.Minor >= minor
}

// Known reports whether the version was actually determined. The zero value is
// unknown, and callers must not read an unknown version as "old" or "new".
func (v ServerVersion) Known() bool { return v.Major != 0 }

// parseServerVersion pulls major and minor out of a TrueNAS release string.
//
// Shapes seen in the wild: "25.10.4", "26.0.0-BETA.2", "TrueNAS-SCALE-24.10.2".
// Everything before the first digit is dropped, then the first two dot-separated
// numbers are taken. Anything that does not yield two numbers is an error, never
// a default, because a wrong version silently selects the wrong wire format.
func parseServerVersion(raw string) (ServerVersion, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ServerVersion{}, fmt.Errorf("server reported an empty version")
	}

	// Skip a leading product prefix such as "TrueNAS-SCALE-".
	start := strings.IndexFunc(s, func(r rune) bool { return r >= '0' && r <= '9' })
	if start < 0 {
		return ServerVersion{}, fmt.Errorf("no version number in %q", raw)
	}
	numeric := s[start:]

	// Stop at the first character that cannot start a numeric segment, so
	// "26.0.0-BETA.2" trims to "26.0.0".
	if cut := strings.IndexAny(numeric, "-+ _"); cut >= 0 {
		numeric = numeric[:cut]
	}

	parts := strings.Split(numeric, ".")
	if len(parts) < 2 {
		return ServerVersion{}, fmt.Errorf("version %q has no minor component", raw)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ServerVersion{}, fmt.Errorf("version %q: major %q is not a number", raw, parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return ServerVersion{}, fmt.Errorf("version %q: minor %q is not a number", raw, parts[1])
	}
	return ServerVersion{Major: major, Minor: minor, Raw: strings.TrimSpace(raw)}, nil
}

// ServerVersion returns the connected server's release, caching it for the life
// of the client. A TrueNAS does not change version mid-apply, so one probe is
// enough and the cost does not repeat per resource.
//
// A probe failure is returned, never swallowed into a default version: choosing
// a wire format from a guessed version is how a "successful" apply writes the
// wrong thing.
func (c *Client) ServerVersion(ctx context.Context) (ServerVersion, error) {
	c.serverVersionMu.Lock()
	cached, err := c.serverVersion, c.serverVersionErr
	c.serverVersionMu.Unlock()
	if cached.Known() {
		return cached, nil
	}
	if err != nil {
		return ServerVersion{}, err
	}

	info, err := c.GetSystemInfo(ctx)
	if err != nil {
		return ServerVersion{}, fmt.Errorf("determining TrueNAS version: %w", err)
	}
	v, perr := parseServerVersion(info.Version)
	if perr != nil {
		perr = fmt.Errorf("determining TrueNAS version: %w", perr)
		c.serverVersionMu.Lock()
		c.serverVersionErr = perr
		c.serverVersionMu.Unlock()
		return ServerVersion{}, perr
	}

	c.serverVersionMu.Lock()
	c.serverVersion = v
	c.serverVersionMu.Unlock()
	return v, nil
}
