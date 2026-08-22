package provider

import (
	"os"
	"strings"
	"testing"
)

// CHANGELOG.md is hand-maintained. It used to be nominally owned by
// changie, but that flow was never completed: no per-PR entry file ever
// reached main, the header template .changie.yaml referenced was never
// added, and no release was ever batched into a version file. Running the
// documented `changie merge` against that state did not fail safe. It
// truncated the file from 227 bullets to 0 while printing an error about
// the missing template.
//
// The tooling is gone, but the failure mode it demonstrated is not
// specific to changie: any command that rewrites CHANGELOG.md from
// sources that do not exist silently empties it, and a changelog is the
// kind of file whose loss nobody notices until a release.
//
// So this asserts the file still looks like a changelog. It is a floor,
// not an exact count, so ordinary edits do not touch it.
const changelogMinimumBullets = 200

func TestChangelogNotTruncated(t *testing.T) {
	raw, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatalf("reading CHANGELOG.md: %v", err)
	}
	body := string(raw)

	if !strings.Contains(body, "## [Unreleased]") {
		t.Error("CHANGELOG.md has no [Unreleased] section. Releasing renames it and " +
			"opens a fresh one; if it is simply gone, something rewrote the file.")
	}

	bullets := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "- ") {
			bullets++
		}
	}
	if bullets < changelogMinimumBullets {
		t.Errorf("CHANGELOG.md is down to %d entries, floor is %d.\n\n"+
			"Entries are only ever added, and released ones stay in the file, so a drop "+
			"means something rewrote it rather than appended to it. Do not lower this "+
			"floor to make the test pass; find what truncated the file.",
			bullets, changelogMinimumBullets)
	}
}
