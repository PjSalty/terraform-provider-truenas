# Releasing terraform-provider-truenas

The release process is **fully autonomous**. This document is the runbook for
the rare case when you need to intervene manually (rollback, emergency fix,
or rotating the maintenance token).

## Normal flow — happy path

There is no `git tag` step. Releases are driven by `CHANGELOG.md`:

1. Open a PR against `dev` with your code change. Add an entry under
   `## [Unreleased]` describing what changed for users.
2. When the change is ready to ship, **bump the heading**: rename
   `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD` and add a fresh empty
   `## [Unreleased]` block above it.
3. Merge the PR (squash) into `dev`.
4. The `promote` job replicates the `dev` tree to `main` and reads the latest
   concrete version from `CHANGELOG.md`. If the resulting tag (`vX.Y.Z`) does
   not yet exist on the remote, it is created and pushed.
5. The mirror replicates the tag to GitHub. The GitHub Actions release
   workflow builds, signs, and uploads release artifacts. The Terraform
   Registry indexes the new version within minutes.

That's it. No manual `git tag`. No manual `terraform-provider-truenas` push.

## Versioning rules

- **Patch (`1.10.x`)** — bug fix, doc fix, dependency bump, no schema change.
- **Minor (`1.x.0`)** — new resource, new optional attribute, new data source.
- **Major (`2.0.0`)** — removed/renamed attribute, changed default that
  breaks state, dropped TrueNAS SCALE version support.

## Rolling back a bad release

If a published version is broken (release-pipeline failure, schema
regression, signing key issue), do this:

```sh
# 1. Delete the tag from GitLab so a future promote can re-publish it cleanly
glab api --method DELETE "projects/16/repository/tags/v1.10.X"

# 2. Delete the GitHub release + tag
gh release delete v1.10.X --repo PjSalty/terraform-provider-truenas \
    --cleanup-tag --yes

# 3. The Terraform Registry caches the version metadata for ~24h. To remove
#    it sooner, open a registry support request — there is no public API for
#    deleting a published provider version.
```

Then fix forward on `dev`, bump `CHANGELOG.md` to the next version, merge.
The auto-tag step picks up the new version and the registry republishes.

## Manual tag (emergency only)

If `promote:to-main` is broken and you need to ship anyway:

```sh
# Run from a clean checkout of main
NEW_VERSION=1.10.X
git tag -a "v${NEW_VERSION}" -m "release: terraform-provider-truenas v${NEW_VERSION}"
git push origin "v${NEW_VERSION}"
git push https://github.com/PjSalty/terraform-provider-truenas.git "v${NEW_VERSION}"
```

The mirror normally handles the second push, but pushing both ways
guarantees the GitHub Actions release workflow fires even during a mirror
outage.

## Token rotation

The `PROMOTE_TOKEN` CI variable is a Project Access Token with
`api + write_repository` scope and Maintainer role. It expires once a year.
Rotation procedure:

```sh
GLAB_TOKEN=<your admin GitLab PAT>
EXPIRY=$(date -d '+1 year' +%Y-%m-%d)

# 1. Issue a new token
NEW=$(curl -sf --header "Private-Token: $GLAB_TOKEN" \
  --header "Content-Type: application/json" \
  --data "{\"name\":\"promote-and-renovate\",\"scopes\":[\"api\",\"write_repository\"],\"access_level\":40,\"expires_at\":\"$EXPIRY\"}" \
  --request POST \
  https://gitlab.salt.saltstice.com/api/v4/projects/16/access_tokens \
  | jq -r .token)

# 2. Update the CI variable
curl -sf --header "Private-Token: $GLAB_TOKEN" \
  --header "Content-Type: application/json" \
  --data "{\"value\":\"$NEW\"}" \
  --request PUT \
  https://gitlab.salt.saltstice.com/api/v4/projects/16/variables/PROMOTE_TOKEN

# 3. Revoke the old token (find its id via GET projects/16/access_tokens)
```

A future improvement would be to wire a scheduled `gomplate`-based job that
rotates this 30 days before expiry, but for now it is a calendar reminder.

## What the maintenance loop looks like

Renovate runs every Monday at 06:00 UTC against `dev`. For each outdated
dependency, it opens an MR. Patch + minor updates auto-merge after CI
passes. Major Go module updates and `terraform-plugin-framework` ecosystem
bumps require human review (they can change schema behaviour).

A typical Monday: 2–6 auto-merged dependency MRs land on `dev`, the promote
job tags `v1.10.(N+1)` if any of them touched module-affecting files, and a
new patch release ships. No human action required.
