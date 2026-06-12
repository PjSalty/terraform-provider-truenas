# wsclient-smoke

Smallest possible Terraform workspace that exercises the
JSON-RPC 2.0 over WebSocket transport against a real TrueNAS
SCALE host.

## What it tests

- The provider can dial `wss://<host>/api/current`.
- The login_with_api_key handshake succeeds.
- A round-trip to `system.info` returns sensible data.
- The websocket connection closes cleanly when Terraform finishes.

## What it does NOT test

- Any resource lifecycle (this workspace is read-only).
- The REST transport (v2.0 ships WebSocket-only; if you need REST
  pin to the v1.x provider line).
- Any pool/dataset/share-specific code path. If those break
  but `system.info` still returns, this workspace will be green —
  the breakage shows up in the per-resource acceptance suite.

## Run

```sh
cd examples/wsclient-smoke
export TF_VAR_truenas_url='https://truenas.example.com'
export TF_VAR_truenas_api_key="$(your-secret-store-fetch-command)"
terraform init
terraform plan
```

Expected outcome:

```
Changes to Outputs:
  + transport_verified = "websocket"
  + truenas_version    = "TrueNAS-SCALE-25.04.x"

Plan: 0 to add, 0 to change, 0 to destroy.
```

## When to use this

- First contact: before pointing a real workspace at a new
  TrueNAS host, run this to prove the WebSocket dial +
  handshake + system.info call all succeed end-to-end.
- Triage: when a per-resource acc test shows unexpected diffs,
  run this first to rule out a connectivity / proxy / TLS issue.
  If this is green, the issue is in the per-resource code path,
  not the wire.
- Upgrade verification: after upgrading TrueNAS SCALE (especially
  the 25.04 → 25.10 jump or any minor with API touch), run this
  to confirm /api/current still accepts the v2.0 provider's
  handshake.

## Failure modes

| Symptom | Likely cause |
| --- | --- |
| `failed to WebSocket dial: ... 404` | TrueNAS host is on SCALE 24.x or older. /api/current isn't exposed; v2.0 requires SCALE 25.04+. Either upgrade SCALE or pin the provider to `~> 1.10`. |
| `failed to WebSocket dial: ... 401` | API key is invalid or revoked. |
| Connection hangs at "Refreshing state..." | Load balancer or proxy in front of TrueNAS isn't passing the WebSocket Upgrade header. Verify with `curl -I -H 'Upgrade: websocket' -H 'Connection: upgrade' https://<host>/api/current` outside of Terraform. |
| Plan succeeds but `truenas_version` is empty | The provider read code path got a successful response with no version field. File an issue. |
