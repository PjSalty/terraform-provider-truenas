package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// authenticate performs the post-dial JSON-RPC handshake that turns an
// anonymous WebSocket into an authenticated session. TrueNAS exposes
// `auth.login_with_api_key` which takes the literal API key string in
// the params array. On success the server returns true; on failure
// either a typed RPCError or a `false` result.
//
// The handshake is non-idempotent (a successful login changes server
// session state), so we do not flag it Idempotent. It also is not
// classified as mutating for the read-only gate's purposes — a read-
// only client still needs to authenticate before it can read.
func (c *Client) authenticate(ctx context.Context) error {
	result, err := c.Call(ctx, "auth.login_with_api_key", []interface{}{c.apiKey}, CallOptions{
		Read:             true, // bypass read-only suffix classifier; auth must succeed even with ReadOnly=true
		disableReconnect: true, // reconnectIfNeeded calls authenticate; re-entering reconnect would deadlock on reconnectMu
	})
	if err != nil {
		return err
	}
	var ok bool
	if err := json.Unmarshal(result, &ok); err != nil {
		// TrueNAS 25.04+ has been observed returning either a bare
		// boolean or an object with a `username` field. Accept either
		// shape; reject anything else as a hard failure.
		var obj map[string]interface{}
		if jsonErr := json.Unmarshal(result, &obj); jsonErr != nil {
			return fmt.Errorf("auth.login_with_api_key: unexpected result shape: %s", string(result))
		}
		return nil
	}
	if !ok {
		return errors.New("auth.login_with_api_key: server returned false (invalid API key?)")
	}
	return nil
}
