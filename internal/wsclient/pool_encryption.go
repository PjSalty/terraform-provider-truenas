package wsclient

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Validation for pool encryption_options.
//
// truenas_pool takes encryption_options as a raw JSON passthrough, which means
// whatever the operator writes reaches middleware unfiltered. PoolCreate embeds
// PoolCreateEncryptionOptions as a typed submodel and the API models are
// ConfigDict(extra="forbid", strict=True), so an unknown key is a hard
// ValidationError on a half-built pool rather than an ignored field.
//
// Two of those keys changed incompatibly in TrueNAS 26.0:
//
//	algorithm     present through 25.10, REMOVED in 26.0 and 27.0
//	pbkdf2iters   floor raised from 100000 (default 350000) to 1300000
//
// A `from_previous` adapter exists upstream that clamps pbkdf2iters, and it
// never runs for us: adapters only apply to a client pinned to an older
// endpoint, and this client dials /api/current.
//
// This is checked client-side, before the call, so the operator gets an
// actionable diagnostic naming the key and the version instead of a pydantic
// traceback surfaced mid-create.

// Minimum pbkdf2iters accepted from TrueNAS 26.0 onward.
const poolMinPBKDF2Iters26 = 1300000

// encryptionOptionKeys is every key PoolCreateEncryptionOptions has ever
// accepted. Anything outside this set is a typo or a hallucinated option and is
// rejected regardless of version, since forwarding it is a guaranteed
// ValidationError.
var encryptionOptionKeys = map[string]bool{
	"generate_key": true,
	"pbkdf2iters":  true,
	"passphrase":   true,
	"key":          true,
	"algorithm":    true, // <= 25.10 only, see below
}

// ValidatePoolEncryptionOptions checks a raw encryption_options map against the
// connected server.
//
// The version is resolved through the client rather than taken as an argument
// so callers cannot forget to probe, and a probe failure is returned rather
// than assumed away: without a version there is no safe branch to take.
func (c *Client) ValidatePoolEncryptionOptions(ctx context.Context, opts map[string]interface{}) error {
	if len(opts) == 0 {
		return nil
	}

	// Unknown keys are wrong on every version; no probe needed.
	var unknown []string
	for k := range opts {
		if !encryptionOptionKeys[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("encryption_options: unknown key(s) %s; TrueNAS accepts only %s",
			quoteList(unknown), quoteList(knownEncryptionOptionKeys()))
	}

	v, err := c.ServerVersion(ctx)
	if err != nil {
		return err
	}
	if !v.AtLeast(26, 0) {
		// 25.10 and older: algorithm is valid and the pbkdf2iters floor is
		// the old one. Nothing further to check.
		return nil
	}

	if _, present := opts["algorithm"]; present {
		return fmt.Errorf("encryption_options: %q was removed in TrueNAS 26.0 and this server reports %s; "+
			"remove the key (AES-256-GCM is the only algorithm on 26.0 and newer)", "algorithm", v)
	}

	if raw, present := opts["pbkdf2iters"]; present {
		n, ok := toInt(raw)
		if !ok {
			return fmt.Errorf("encryption_options: %q must be a number, got %T", "pbkdf2iters", raw)
		}
		// 0 means "use the server default" in the pre-26 models, and the
		// 26 default already satisfies the floor, so only a positive value
		// below the floor is a problem.
		if n > 0 && n < poolMinPBKDF2Iters26 {
			return fmt.Errorf("encryption_options: %q is %d, but TrueNAS 26.0 raised the minimum to %d "+
				"and this server reports %s; raise it or drop the key to take the server default",
				"pbkdf2iters", n, poolMinPBKDF2Iters26, v)
		}
	}
	return nil
}

// toInt accepts the shapes a JSON number can decode into.
func toInt(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}

func knownEncryptionOptionKeys() []string {
	keys := make([]string, 0, len(encryptionOptionKeys))
	for k := range encryptionOptionKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func quoteList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, s := range items {
		quoted = append(quoted, fmt.Sprintf("%q", s))
	}
	return strings.Join(quoted, ", ")
}
