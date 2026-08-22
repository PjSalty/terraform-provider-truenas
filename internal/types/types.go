// Package types holds the request/response struct types the WebSocket
// JSON-RPC client (internal/wsclient) marshals to and from, and that
// resources in internal/resources/ build directly.
//
// Why a separate package: it lets resource code refer to
// types.DatasetResponse without importing a transport, and it kept the
// two transports from importing each other during the v2.0 migration
// off REST. That migration is finished. internal/client, the REST
// client these types were once shared with, no longer exists, and
// wsclient is the only transport.
package types
