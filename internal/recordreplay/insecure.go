package recordreplay

import (
	"crypto/tls"
	"net/http"
)

// insecureHTTPClient is the upstream forwarder for the Recorder.
// The test TrueNAS ships with a self-signed cert; production
// recordings should use a configured TLS bundle (CI pin).
var insecureHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test fixture only
	},
}
