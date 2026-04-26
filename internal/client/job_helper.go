package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// waitIfJobResponse inspects a TrueNAS API response body for a numeric
// job ID and, if found, waits for that job to complete. Any response
// that does NOT parse as a JSON integer is treated as "no async job —
// the operation completed synchronously" and returns nil.
//
// This dual-response pattern is documented TrueNAS middlewared
// behavior: certain endpoints (DELETE /app/id/{id}, POST /pool/id/{id}/export,
// GET /catalog/sync, DELETE /certificate/id/{id}) return a plain integer
// job ID when they kick off async work, and return null / an empty
// object / an arbitrary sentinel when the work completed synchronously.
// Previous code paths open-coded the pattern with an `err != nil →
// return nil` structure that looked like a bug to the nilerr linter;
// centralizing it here lets us declare the intent once and silence
// the linter in exactly one place with a clear rationale.
//
// opLabel is used in the error message if the WaitForJob call itself
// fails, e.g. "deleting app" → "waiting for deleting app job: ...".
func (c *Client) waitIfJobResponse(ctx context.Context, resp []byte, opLabel string) error {
	var jobID int
	// Intentional: a non-integer response body is the "no job, sync op"
	// signal from middlewared, NOT an error to propagate. Returning nil
	// here is correct — see the function-level comment for details.
	//nolint:nilerr // dual-response API contract; see doc comment
	if err := json.Unmarshal(resp, &jobID); err != nil {
		return nil
	}
	if _, err := c.WaitForJob(ctx, jobID); err != nil {
		return fmt.Errorf("waiting for %s job: %w", opLabel, err)
	}
	return nil
}
