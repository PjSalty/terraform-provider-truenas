# Releasing terraform-provider-truenas

Releases are cut by hand from `main` and published by GitHub Actions. This
document is the runbook.

## Normal flow, happy path

1. Open an MR against `main` with your code change. Add an entry under
   `## [Unreleased]` in `CHANGELOG.md` describing what changed for users.
2. When the change is ready to ship, open a release MR that renames
   `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD` and adds a fresh empty
   `## [Unreleased]` block above it. Nothing else goes in that MR.
3. Merge it.
4. From a clean checkout of `main` at that merge commit, run `make prod-ready`.
   It has to print `safe to tag`; that target is the gate, and it exercises
   the static analysis, the docs and examples ratchets, and the acceptance
   coverage ratchet.
5. Tag and push to both remotes:

   ```sh
   NEW_VERSION=X.Y.Z
   git tag -a "v${NEW_VERSION}" -m "release: terraform-provider-truenas v${NEW_VERSION}"
   git push origin "v${NEW_VERSION}"
   git push https://github.com/PjSalty/terraform-provider-truenas.git "v${NEW_VERSION}"
   ```

   The mirror normally replicates the tag on its own. Pushing both ways
   guarantees the GitHub Actions release workflow fires even during a mirror
   outage, and pushing a tag that already exists is a no-op.
6. `.github/workflows/release.yml` fires on `v*`, runs goreleaser, signs with
   the GPG key when `GPG_PRIVATE_KEY` is set, and uploads the artifacts. The
   Terraform Registry indexes the new version within minutes.
7. Check the release actually contains something before calling it done:

   ```sh
   gh release view "v${NEW_VERSION}" --json assets --jq '.assets[].name'
   ```

   A tag can point at a partial tree, and a release can exist with no
   binaries. Verify the assets, not just that the tag resolves.

There is no `dev` branch and no promote job. An earlier version of this
document described a `dev` -> `main` promote that read the version out of
`CHANGELOG.md` and tagged automatically. It does not exist in
`.gitlab-ci.yml`, and every tag from v2.1.0 onward was made by hand, so the
runbook was describing a process nobody could follow.

## Versioning rules

- **Patch (`X.Y.Z+1`)**, bug fix, doc fix, dependency bump, no schema change.
- **Minor (`X.Y+1.0`)**, new resource, new optional attribute, new data
  source. A validator that stops accepting a value the API never accepted is
  minor: no configuration that worked can break.
- **Major (`X+1.0.0`)**, removed or renamed attribute, changed default that
  breaks state, dropped TrueNAS SCALE version support. A rename is major even
  when the resource it belongs to was non-functional, because the old
  attribute name in a practitioner's configuration becomes a hard
  "Unsupported argument" error on upgrade.

## Rolling back a bad release

If a published version is broken (release-pipeline failure, schema
regression, signing key issue), do this:

```sh
# 1. Delete the tag from GitLab so a future release can re-publish it cleanly
glab api --method DELETE "projects/16/repository/tags/vX.Y.Z"

# 2. Delete the GitHub release + tag
gh release delete vX.Y.Z --repo PjSalty/terraform-provider-truenas \
    --cleanup-tag --yes

# 3. The Terraform Registry caches the version metadata for ~24h. To remove
#    it sooner, open a registry support request, there is no public API for
#    deleting a published provider version.
```

Then fix forward on `main`, bump `CHANGELOG.md` to the next version, and tag
again.

## PROMOTE_TOKEN

The `PROMOTE_TOKEN` CI variable is a Project Access Token with
`api + write_repository` scope and Maintainer role. **Nothing reads it.** It
was created for the promote job described above, and that job does not exist,
so the token is an unused credential with write access to the repository.

Either revoke it, or wire the promote job it was made for. Leaving it is the
one option that has no upside. Rotation, if it is kept:

```sh
GLAB_TOKEN=<your admin GitLab PAT>
EXPIRY=$(date -d '+1 year' +%Y-%m-%d)

# 1. Issue a new token
NEW=$(curl -sf --header "Private-Token: $GLAB_TOKEN" \
  --header "Content-Type: application/json" \
  --data "{\"name\":\"promote-and-renovate\",\"scopes\":[\"api\",\"write_repository\"],\"access_level\":40,\"expires_at\":\"$EXPIRY\"}" \
  --request POST \
  https://gitlab.example.com/api/v4/projects/16/access_tokens \
  | jq -r .token)

# 2. Update the CI variable
curl -sf --header "Private-Token: $GLAB_TOKEN" \
  --header "Content-Type: application/json" \
  --data "{\"value\":\"$NEW\"}" \
  --request PUT \
  https://gitlab.example.com/api/v4/projects/16/variables/PROMOTE_TOKEN

# 3. Revoke the old token (find its id via GET projects/16/access_tokens)
```

## What the maintenance loop looks like

Renovate runs every Monday before 06:00 against the default branch, `main`
(`renovate.json` sets no `baseBranches`, so the default is used). For each
outdated dependency it opens an MR. Patch and minor updates auto-merge after
CI passes. Major Go module updates and `terraform-plugin-framework` ecosystem
bumps require human review, because they can change schema behaviour.

Those merges accumulate under `## [Unreleased]`. Shipping them is the release
MR in step 2, which is a deliberate act, not a side effect of a dependency
bump landing.
