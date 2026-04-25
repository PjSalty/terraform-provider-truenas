<!--
Thank you for contributing to terraform-provider-truenas!

Items marked [REQUIRED] must be present before this PR can be reviewed.
-->

## Summary
<!-- [REQUIRED] 1-3 sentence description of the change and its motivation. -->

## Type of change
<!-- [REQUIRED] Check one. -->
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New resource (new `truenas_*` resource)
- [ ] New data source (new `data.truenas_*` data source)
- [ ] Enhancement to an existing resource/data source
- [ ] Provider-level change (client, retry, timeouts, schema)
- [ ] Documentation only
- [ ] CI / tooling / build
- [ ] Breaking change (requires major version bump)

## Resources / data sources affected
<!-- [REQUIRED] List every `truenas_*` resource or data source this PR touches. -->

## Checklist
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes
- [ ] `go test -cover ./...` maintains 100.0% on every package
- [ ] `golangci-lint run ./...` passes with zero findings
- [ ] `govulncheck ./...` reports no new vulnerabilities
- [ ] `tfplugindocs validate` passes
- [ ] I ran the relevant `TF_ACC` acceptance tests against a real TrueNAS SCALE
      test VM (not prod) and recorded the results below
- [ ] CHANGELOG.md `[Unreleased]` section updated
- [ ] Docs regenerated (`make docs` or `tfplugindocs generate`)
- [ ] Examples in `examples/resources/<name>/` updated if the schema changed
- [ ] Sensitive fields marked with `Sensitive: true`
- [ ] New validators added to schema where applicable

## Acceptance test results
<!-- Paste the output of the acceptance test run, or "N/A" for docs/CI-only PRs. -->

```
```

## Breaking change notice
<!-- If this is a breaking change, describe what users need to do to migrate. -->

## Related issues
<!-- Closes #N, Refs #M, etc. -->
