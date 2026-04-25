# Security Policy

## Supported Versions

Only the latest minor release line receives security updates. Older releases
should upgrade to the current line before reporting issues against them.

| Version | Supported |
|---------|-----------|
| 1.x     | ✅        |
| < 1.0   | ❌        |

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

If you believe you have found a security issue in this provider (for example:
credential leakage, incorrect TLS verification, privilege escalation via the
TrueNAS API, unsafe handling of sensitive attributes, or a dependency
vulnerability not yet covered by `govulncheck`), please report it privately:

- GitHub: use [private vulnerability reporting](https://github.com/PjSalty/terraform-provider-truenas/security/advisories/new)
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

- The provider binary and its client library (`internal/client`,
  `internal/resources`, `internal/datasources`).
- How the provider handles API keys, SOPS-encrypted credentials, and any
  sensitive Terraform attributes.
- TLS behavior, including the `insecure_skip_verify` escape hatch.
- Dependency vulnerabilities reachable from the provider's runtime surface.

Out of scope:

- Issues in TrueNAS SCALE itself — report those to the upstream iXsystems
  project.
- Issues in Terraform or `terraform-plugin-framework` — report those to
  HashiCorp.
- Issues requiring privileged access to an already-compromised host.

## Dependency Vulnerabilities

`govulncheck` runs in CI on every commit. If you see a new advisory land
before the next scheduled CI run, open a confidential issue or submit a PR
that bumps the affected module.
