package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// valueSetDrift compares a provider OneOf against the set the TrueNAS API
// actually accepts, recorded in testdata/value_sets.json.
//
// scripts/api-drift.sh catches a METHOD that upstream removed. Nothing caught a
// VALUE that upstream accepts and the provider refuses, which is how six
// attributes drifted: truenas_dataset rejected every ZSTD level and every
// recordsize above 1M, truenas_smb_config allowed six charsets of which four
// were spelled the way Samba writes them rather than the way the API does, and
// truenas_service rejected two services that exist on the box.
//
// The recorded set is checked in rather than fetched, so this runs with no
// TrueNAS and no network. Refresh it with `make value-sets` against a live box
// when a new TrueNAS lands.
type valueSet struct {
	Resource  string   `json:"resource"`
	Attribute string   `json:"attribute"`
	File      string   `json:"file"`
	Source    string   `json:"source"`
	Values    []string `json:"values"`
	// AllowExtra lists values the provider may accept that the API set does
	// not contain, each with a reason. Used where upstream types the field as
	// a free string and the provider keeps a wider convenience list.
	AllowExtra map[string]string `json:"allow_extra,omitempty"`
}

func TestProviderValueSetsMatchAPI(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "internal", "provider", "testdata", "value_sets.json"))
	if err != nil {
		t.Fatalf("reading recorded value sets: %v", err)
	}
	var recorded []valueSet
	if err := json.Unmarshal(raw, &recorded); err != nil {
		t.Fatalf("decoding recorded value sets: %v", err)
	}
	if len(recorded) == 0 {
		t.Fatal("no value sets recorded; the file exists but asserts nothing")
	}

	for _, vs := range recorded {
		t.Run(vs.Resource+"."+vs.Attribute, func(t *testing.T) {
			got, err := oneOfValues(filepath.Join(root, vs.File), vs.Attribute)
			if err != nil {
				t.Fatalf("%v", err)
			}
			api := map[string]bool{}
			for _, v := range vs.Values {
				api[v] = true
			}
			provider := map[string]bool{}
			for _, v := range got {
				provider[v] = true
			}

			var missing []string
			for v := range api {
				if !provider[v] {
					missing = append(missing, v)
				}
			}
			var extra []string
			for v := range provider {
				if !api[v] {
					if _, ok := vs.AllowExtra[v]; ok {
						continue
					}
					extra = append(extra, v)
				}
			}
			sort.Strings(missing)
			sort.Strings(extra)

			if len(missing) > 0 {
				t.Errorf("%s.%s rejects %d value(s) the API accepts, so they cannot be "+
					"configured at all: %v\n(API set from %s)",
					vs.Resource, vs.Attribute, len(missing), missing, vs.Source)
			}
			if len(extra) > 0 {
				t.Errorf("%s.%s accepts %d value(s) the API does not, so they pass plan "+
					"and fail at apply: %v\nIf one is deliberate, add it to allow_extra "+
					"with a reason.\n(API set from %s)",
					vs.Resource, vs.Attribute, len(extra), extra, vs.Source)
			}
		})
	}
}

// oneOfValues reads the string literals of the OneOf validator attached to an
// attribute. It parses the source text rather than the built schema because a
// validator does not expose its accepted set through the framework interface.
func oneOfValues(path, attr string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := string(src)
	i := strings.Index(s, fmt.Sprintf("%q: schema.", attr))
	if i < 0 {
		return nil, fmt.Errorf("%s: attribute %q not found", path, attr)
	}
	j := strings.Index(s[i:], "stringvalidator.OneOf(")
	if j < 0 {
		return nil, fmt.Errorf("%s: attribute %q has no stringvalidator.OneOf", path, attr)
	}
	j += i + len("stringvalidator.OneOf(") - 1

	depth, k := 0, j
	for ; k < len(s); k++ {
		switch s[k] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 {
			break
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("%s: unbalanced OneOf for %q", path, attr)
	}

	body := s[j : k+1]
	// A variadic expansion such as OneOf(smbPurposes...) has no literals to
	// read. Report that rather than returning an empty set, which a caller
	// would otherwise read as "this attribute accepts nothing".
	if strings.Contains(body, "...") {
		return nil, fmt.Errorf("%s: %q uses a variadic OneOf, its values are not literals", path, attr)
	}

	var out []string
	for _, line := range strings.Split(body, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		for _, m := range strings.Split(line, `"`)[1:] {
			_ = m
		}
		parts := strings.Split(line, `"`)
		for n := 1; n < len(parts); n += 2 {
			out = append(out, parts[n])
		}
	}
	// An empty accepted value cannot be enumerated in prose, so it is dropped
	// here and callers compare only the nameable ones.
	kept := out[:0]
	for _, v := range out {
		if v != "" {
			kept = append(kept, v)
		}
	}
	sort.Strings(kept)
	if len(kept) == 0 {
		return nil, fmt.Errorf("%s: %q has no literal OneOf values", path, attr)
	}
	return kept, nil
}
