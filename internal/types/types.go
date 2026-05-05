// Package types holds request/response struct types shared between the
// REST client (internal/client) and the WebSocket JSON-RPC client
// (internal/wsclient). Resources in internal/resources/ reach into these
// types directly so the same Terraform schema marshals to and from
// either transport without code-level transport awareness.
//
// Why a separate package: while internal/client owns these types today,
// adding the wsclient transport in v2.0 means both packages need to
// produce and consume the same shapes. Putting them here lets each
// transport import without circular dependency on the other, and lets
// resource code refer to types.DatasetResponse without picking a
// transport.
//
// Migration policy: types move here per-resource as that resource is
// migrated to wsclient. Until a resource is migrated, its types stay
// in internal/client/. After v2.0 (Phase 5) deletes internal/client/,
// any remaining types move here in one final sweep.
package types
