# Contributing to terraform-provider-truenas

Thanks for contributing! This provider follows the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
conventions and the quality bar established by major HashiCorp-maintained
Terraform providers.

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before participating.
For security issues, follow [SECURITY.md](SECURITY.md) — **do not open a
public issue for vulnerabilities**.

## Table of contents

- [Requirements](#requirements)
- [Quickstart](#quickstart)
- [Repository layout](#repository-layout)
- [Development workflow](#development-workflow)
- [Testing](#testing)
- [Adding a new resource](#adding-a-new-resource)
- [Adding a new data source](#adding-a-new-data-source)
- [Validators, plan modifiers, and flex helpers](#validators-plan-modifiers-and-flex-helpers)
- [Documentation](#documentation)
- [Changelog entries](#changelog-entries)
- [Sensitive attribute policy](#sensitive-attribute-policy)
- [Code quality gates](#code-quality-gates)
- [Release process](#release-process)

## Requirements

- [Go](https://go.dev/) 1.25+
- [Terraform](https://www.terraform.io/downloads) 1.8+ (for acceptance tests)
- [golangci-lint](https://golangci-lint.run/) v1.62+
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs)
- [goreleaser](https://goreleaser.com/) v2.15+
- [pre-commit](https://pre-commit.com/) (recommended)
- A TrueNAS SCALE 25.10+ test VM (for acceptance tests — **never** your
  production TrueNAS)

## Quickstart

```sh
git clone https://github.com/PjSalty/terraform-provider-truenas
cd terraform-provider-truenas
go mod download
pre-commit install       # optional but recommended
make build               # produces ./terraform-provider-truenas
make test                # unit tests, httptest-mocked, no live infra
make lint                # golangci-lint
```

## Repository layout

```
terraform-provider-truenas/
├── main.go                        # Provider entry point
├── main_test.go
├── internal/
│   ├── acctest/                   # Shared acceptance test helpers
│   ├── client/                    # TrueNAS REST client (per-domain files)
│   ├── datasources/               # Terraform data sources
│   ├── flex/                      # Framework<->Go type conversion helpers
│   ├── fwresource/                # Framework resource base helpers (Configure etc.)
│   ├── planmodifiers/             # Reusable plan modifiers
│   ├── provider/                  # Provider registration + acc_*_test.go
│   ├── resources/                 # Resource implementations + unit tests
│   ├── sweep/                     # Acceptance test sweeper infrastructure
│   └── validators/                # Custom attribute validators
├── docs/                          # Terraform Registry docs (tfplugindocs)
│   ├── data-sources/
│   ├── guides/
│   └── resources/
├── examples/                      # Runnable example configs per resource
├── testdata/                      # Test fixtures incl. fuzz corpus
│   └── fuzz/                      # Go native fuzz corpus
├── tools/
│   └── tools.go                   # Dev-only tooling pins (build tag: tools)
├── .changelog/                    # changie per-PR changelog entries
├── .github/                       # GitHub workflows, issue/PR templates
├── .goreleaser.yml                # Multi-platform release config
├── .golangci.yml                  # 13-linter config
├── .pre-commit-config.yaml        # Local pre-commit hooks
├── Makefile                       # Build/test/lint targets
└── renovate.json                  # Dependency automation
```

## Development workflow

1. **Open an issue first** for anything non-trivial so we can agree on scope.
2. **Create a feature branch** off `main`: `git checkout -b feat/my-thing`.
3. **Make atomic commits** following the
   [Conventional Commits](https://www.conventionalcommits.org/) spec:
   - `feat(cloudsync_credential): add Backblaze B2 provider type`
   - `fix(certificate): normalize ENOENT job error to 404`
   - `test(vm): cover bootloader UEFI cross-attribute constraint`
4. **Run all gates locally** before pushing (see below).
5. **Add a `.changelog/unreleased/` entry** via `changie new`.
6. **Open a pull request** against `main`, fill out the PR template.
7. CI must be green before review. **No exceptions, no `allow_failure: true`
   shortcuts.**

## Testing

### Unit tests

Unit tests use `net/http/httptest` to mock the TrueNAS API. They run without
any live infrastructure and are the fastest feedback loop:

```sh
make test                          # all unit tests with race detector
go test -cover ./internal/...      # with coverage
go test -run TestCreateDataset ./internal/client/  # single test
```

**Coverage requirement**: **100.0%** on every package. CI enforces this as a
gate. If a new branch genuinely cannot be tested, delete the defensive code
instead — don't suppress coverage.

### Acceptance tests

Acceptance tests create real resources against a running TrueNAS SCALE
instance. They are guarded by `TF_ACC=1` so they never run accidentally.

```sh
export TRUENAS_URL=https://test-truenas.example.com
export TRUENAS_API_KEY=1-xxxxxxxxxxxx
export TRUENAS_TEST_POOL=test          # default pool for datasets
export TRUENAS_INSECURE_SKIP_VERIFY=1  # if test VM has a self-signed cert

make testacc
# or target a single resource:
TF_ACC=1 go test -v -run TestAccDatasetResource_basic ./internal/provider/
```

**CRITICAL**: Never run acceptance tests against a production TrueNAS.
Use a dedicated expendable test VM only.

Every new resource must have the full acceptance-test triad:

- `TestAcc<Res>Resource_basic` — happy path create/read/import/destroy
- `TestAcc<Res>Resource_update` — verify Update without replacement
- `TestAcc<Res>Resource_disappears` — out-of-band delete, provider recovers

Singleton config resources use `_basic` + `_update` (no `_disappears` — you
can't delete a singleton).

### Fuzz tests

Fuzz targets live next to unit tests. Short smoke run:

```sh
go test -run='^$' -fuzz='^FuzzParseRetryAfter$' -fuzztime=30s ./internal/client/
```

Long campaign (local only, until ctrl-c):

```sh
go test -run='^$' -fuzz='^FuzzNormalizeJSON$' -fuzztime=1h ./internal/resources/
```

Regression inputs found by fuzz runs are committed under
`testdata/fuzz/<FuzzName>/` and automatically replay as unit tests on every
subsequent `go test` run.

### Benchmarks

```sh
go test -run='^$' -bench=. -benchtime=5s ./internal/...
```

## Adding a new resource

### Quick scaffold with `skaff`

```sh
go run ./cmd/skaff resource my_new_thing
go run ./cmd/skaff datasource my_new_thing
```

This creates skeleton files under `internal/resources/`, `internal/client/`,
`docs/resources/`, and `examples/resources/truenas_my_new_thing/`. Fill in
the schema, CRUD handlers, and client methods, then follow the manual
checklist below.

### Manual checklist

1. **Check the TrueNAS SCALE API reference** for the endpoint:
   <https://www.truenas.com/docs/scale/scaleapireference/>
2. **Add client methods** in `internal/client/<name>.go`:
   - `Get<Name>(ctx, id)`, `Create<Name>(ctx, req)`, `Update<Name>(ctx, id, req)`, `Delete<Name>(ctx, id)`
   - Unit tests in `internal/client/<name>_test.go` covering 200/404/500/invalid-JSON paths.
3. **Add the resource** in `internal/resources/<name>.go`:
   - Implement `resource.Resource`, `resource.ResourceWithImportState`, and
     optionally `resource.ResourceWithModifyPlan` for cross-attribute validation.
   - Use `internal/fwresource.ConfigureClient` in `Configure`.
   - Add a `timeouts.Block` with Create/Read/Update/Delete defaults.
   - Mark credential-bearing attributes with `Sensitive: true`.
   - Apply validators from `internal/validators` and plan modifiers from
     `internal/planmodifiers` where applicable.
4. **Register** the resource in `internal/provider/provider.go` under
   `Resources()`.
5. **Add matching data source** in `internal/datasources/<name>.go` — every
   resource should have a matching data source.
6. **Add tests**:
   - Unit tests for schema/metadata/configure/CRUD in
     `internal/resources/<name>_test.go`.
   - Acceptance test triad in `internal/provider/acc_<name>_test.go`.
   - Data source test in `internal/datasources/<name>_test.go`.
7. **Add sweeper** in `internal/provider/sweeper_test.go` so abandoned test
   fixtures get cleaned up (`TF_ACC=1 go test -sweep=all ./internal/provider/`).
8. **Write docs**:
   - `docs/resources/<name>.md` (frontmatter, Example Usage, Argument
     Reference, Attribute Reference, Timeouts, Import).
   - `docs/data-sources/<name>.md` if applicable.
9. **Add example**: `examples/resources/truenas_<name>/resource.tf` +
   `examples/resources/truenas_<name>/import.sh`.
10. **Add changelog entry** via `changie new`.
11. **Run all gates** (see below) until everything is green.

## Adding a new data source

Follow steps 1, 2 (if client methods don't already exist), and 6-11 above,
but targeted at `internal/datasources/` instead of `internal/resources/`.
Data sources are read-only — no Create/Update/Delete.

## Validators, plan modifiers, and flex helpers

- **`internal/validators`** — reusable attribute validators:
  `ZFSPath()`, `IPOrCIDR()`, `HostOrIP()`, `CompressionAlgorithm()`.
- **`internal/planmodifiers`** — framework plan modifiers:
  `RequiresReplaceIfChanged()`.
- **`internal/flex`** — framework<->Go type conversion helpers:
  `StringPointerValue`, `Int64PointerValue`, `StringListValue`, etc. Prefer
  these over hand-rolled conversions.
- **`internal/fwresource`** — framework resource base helpers:
  `ConfigureClient` for the `Configure` boilerplate.

## Documentation

Documentation is regenerated from schema via `tfplugindocs`:

```sh
make docs                          # regenerate all resource/data-source docs
tfplugindocs validate              # verify registry format is correct
```

Hand-written guides live under `docs/guides/`. Do not commit generated docs
that conflict with your changes — always regenerate after schema edits.

## Changelog entries

We use [changie](https://changie.dev/) for per-PR changelog entries:

```sh
go install github.com/miniscruff/changie@latest
changie new                        # interactive prompt, writes .changelog/unreleased/*.yaml
```

Commit the resulting file as part of your PR. At release time, a maintainer
runs `changie batch vX.Y.Z && changie merge` to fold entries into
`CHANGELOG.md`.

## Sensitive attribute policy

Any attribute carrying credentials, tokens, keys, or passphrases **must** be
marked `Sensitive: true` in the schema. This includes:

- Passwords (`password`, `secret`, `pass`)
- API keys, tokens, passphrases
- TLS private keys
- Kerberos keytabs (even though they're base64-encoded binary)
- Provider-credential JSON blobs (`credentials_json`, `attributes_json`)
- DHCHAP keys, pre-shared keys

If in doubt, mark it sensitive. The sensitivity audit job in CI will flag
credential-looking attribute names that lack `Sensitive: true`.

## Code quality gates

Every commit must pass:

| Gate | Command | Requirement |
|------|---------|-------------|
| Format | `gofmt -l .` | zero output |
| Vet | `go vet ./...` | exit 0 |
| Unit tests | `go test -race ./...` | all pass |
| **Coverage** | `go test -cover ./...` | **100.0%** every package |
| Lint | `golangci-lint run ./...` | zero findings |
| Vulnerabilities | `govulncheck ./...` | "No vulnerabilities found" |
| Docs | `tfplugindocs validate` | exit 0 |
| Release | `goreleaser check` | exit 0 |

CI enforces all of these. Pre-commit hooks catch most issues locally.

## Release process

Releases are cut by tagging `vX.Y.Z` on `main`:

```sh
changie batch vX.Y.Z
changie merge
git add CHANGELOG.md .changelog/
git commit -m "chore(release): vX.Y.Z"
git tag vX.Y.Z
git push origin main vX.Y.Z
```

CI picks up the tag and runs `goreleaser release`, which builds for 14
platform targets (linux/darwin/windows/freebsd × amd64/arm64/arm6/arm7/386),
generates an SBOM via syft, signs the checksum file with GPG, and publishes
the release to GitHub.

See [`.goreleaser.yml`](.goreleaser.yml) for the full release configuration.

## Getting help

- Open an issue with the `question` label.
- Check existing issues and MRs before opening a new one.
- For security issues, follow [SECURITY.md](SECURITY.md) — do not disclose
  publicly.
