# Changelog entries

This directory holds individual changelog entries per pull request, managed
by [changie](https://changie.dev/). Each PR that changes user-visible behavior
must include a changelog entry committed to this directory (or to
`.changelog/unreleased/`).

## Adding an entry

```bash
# Install changie once:
go install github.com/miniscruff/changie@latest

# From repo root:
changie new
```

`changie new` will prompt for the kind (Added / Changed / Deprecated / Removed
/ Fixed / Security) and a one-line summary, then write a YAML file to
`.changelog/unreleased/`.

## Releasing

At release time:

```bash
changie batch v1.2.3
changie merge
```

`changie batch` collects all `unreleased/*.yaml` entries into a single
`v1.2.3.md` file under `.changelog/`. `changie merge` rewrites the top-level
`CHANGELOG.md` from those batched files. Both operations are idempotent and
reversible — only the final merged `CHANGELOG.md` is shipped in releases.

## Configuration

The changie configuration lives in `.changie.yaml` at the repo root.

## Why this exists

Centralized changelog files are a constant source of merge conflicts when
multiple PRs land in parallel. Per-PR entry files avoid the conflicts entirely
and give reviewers a small, focused diff to read.
