# Security Policy

## Supported Versions

The current major line receives full security support. The previous line
receives security fixes only, for users pinned to TrueNAS SCALE versions
the current line does not fully support (24.04 and 25.04).

| Version | Supported |
|---------|-----------|
| 2.x     | ✅ full support |
| 1.10.x  | ⚠️ security fixes only (SCALE 24.04 / 25.04 users) |
| < 1.10  | ❌        |

1.10.x security fixes are cut from the latest v1.10.x tag as needed.
Once TrueNAS SCALE 25.04 leaves upstream support, the 1.x line is
retired with it.

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

If you believe you have found a security issue in this provider (for example:
credential leakage, incorrect TLS verification, privilege escalation via the
TrueNAS API, unsafe handling of sensitive attributes, or a dependency
vulnerability not yet covered by `govulncheck`), please report it privately:

- GitHub: open a private security advisory at
  <https://github.com/PjSalty/terraform-provider-truenas/security/advisories/new>
- Email: security@saltstice.com

Include in your report:

1. A clear description of the issue and its impact.
2. The version of the provider and TrueNAS SCALE you observed it on.
3. A minimal Terraform configuration or reproduction steps.
4. If applicable, a proof-of-concept or the specific field/endpoint affected.

You will receive acknowledgement within **5 business days**. We will work with
you on a coordinated disclosure timeline and credit you in the release notes
if desired.

## Scope

In scope:

- The provider binary and its client library (`internal/wsclient`,
  `internal/resources`, `internal/datasources`).
- How the provider handles API keys, SOPS-encrypted credentials, and any
  sensitive Terraform attributes.
- TLS behavior, including the `insecure_skip_verify` escape hatch.
- Dependency vulnerabilities reachable from the provider's runtime surface.

Out of scope:

- Issues in TrueNAS SCALE itself, report those to the upstream iXsystems
  project.
- Issues in Terraform or `terraform-plugin-framework`, report those to
  HashiCorp.
- Issues requiring privileged access to an already-compromised host.

## Dependency Vulnerabilities

`govulncheck` runs in CI on every commit. If you see a new advisory land
before the next scheduled CI run, open a confidential issue or submit a PR
that bumps the affected module.

## Verifying release artifacts

Every GitHub release ships a GPG-signed `SHA256SUMS` file. The signing
public key is committed to this repository at
[`docs/gpg-public-key.asc`](docs/gpg-public-key.asc) and is also registered
with the Terraform Registry so `terraform init` verifies provider downloads
automatically.

To verify a release manually:

```bash
# Import the signing key
gpg --import docs/gpg-public-key.asc

# Confirm fingerprint matches the one published here:
#   29A6 D319 E411 670F 561E  2B9C EC8F 6B9D 7DB7 49E7
gpg --fingerprint releases@saltstice.com

# Download SHA256SUMS + .sig from the GitHub release, then:
gpg --verify terraform-provider-truenas_<version>_SHA256SUMS.sig \
            terraform-provider-truenas_<version>_SHA256SUMS

# Verify each binary against its checksum:
sha256sum -c terraform-provider-truenas_<version>_SHA256SUMS \
  --ignore-missing
```

If the fingerprint shown by `gpg --fingerprint` does not match the value
above, **do not trust the artifacts**. Open a security advisory immediately.
