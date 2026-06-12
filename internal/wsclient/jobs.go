package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// JobState is the terminal or in-flight state reported by core.get_jobs
// for a TrueNAS long-running operation.
type JobState string

const (
	// JobStateRunning is the in-flight state.
	JobStateRunning JobState = "RUNNING"
	// JobStateWaiting is the queued-but-not-started state.
	JobStateWaiting JobState = "WAITING"
	// JobStateSuccess is the terminal happy state.
	JobStateSuccess JobState = "SUCCESS"
	// JobStateFailed is a terminal error state with a descriptive message.
	JobStateFailed JobState = "FAILED"
	// JobStateAborted is a terminal cancellation state.
	JobStateAborted JobState = "ABORTED"
)

// jobInfo is the partial shape of a core.get_jobs entry that we care
// about. The full server-side struct is much larger; everything not
// listed here is captured into Result/Error as RawMessage so caller-
// specific job types can pull additional fields via a follow-on
// json.Unmarshal.
type jobInfo struct {
	ID       int64           `json:"id"`
	State    JobState        `json:"state"`
	Progress json.RawMessage `json:"progress,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// CallJob invokes a JSON-RPC method that the server runs as a long-
// running job, then polls core.get_jobs until it reaches a terminal
// state. The returned RawMessage is the job's `result` field on
// success; on FAILED or ABORTED the call returns an error carrying the
// job's `error` text.
//
// pollInterval bounds the wait between status checks; zero falls back
// to 2s. Total time is bounded by the caller's ctx deadline.
//
// CallJob is non-idempotent at the call() layer (the initial method
// invocation runs once); retry on connection loss happens transparently
// during the polling phase since core.get_jobs IS idempotent.
func (c *Client) CallJob(ctx context.Context, method string, params []interface{}, opts CallOptions, pollInterval time.Duration) (json.RawMessage, error) {
	// Force-mark as a job; opts.Job semantics are advisory only at this
	// layer (used by per-resource code to know which methods need
	// CallJob), so we ignore opts.Job here and always poll.
	idResult, err := c.Call(ctx, method, params, opts)
	if err != nil {
		return nil, err
	}

	var jobID int64
	if err := json.Unmarshal(idResult, &jobID); err != nil {
		return nil, fmt.Errorf("CallJob %s: server did not return a job ID (got %s): %w", method, string(idResult), err)
	}

	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	// The top of this loop intentionally has no ctx.Err() check: the
	// underlying GetJob -> Call path observes ctx cancellation via
	// the per-call select, and sleepCtx at the bottom catches the
	// sleep-window cancel. A redundant top check would be untestable
	// without race-prone timing setups and would not change behavior.
	for {
		info, err := c.GetJob(ctx, jobID)
		if err != nil {
			return nil, err
		}

		switch info.State {
		case JobStateSuccess:
			return info.Result, nil
		case JobStateFailed:
			return nil, fmt.Errorf("CallJob %s (id=%d) failed: %s", method, jobID, redactMessage(info.Error))
		case JobStateAborted:
			return nil, fmt.Errorf("CallJob %s (id=%d) aborted", method, jobID)
		case JobStateRunning, JobStateWaiting:
			// keep polling
		default:
			return nil, fmt.Errorf("CallJob %s (id=%d): unknown job state %q", method, jobID, info.State)
		}

		if err := sleepCtx(ctx, pollInterval); err != nil {
			return nil, err
		}
	}
}

// GetJob returns the latest server-side state for a single job ID,
// mirroring the REST client's GET /core/get_jobs?id=N. Idempotent -
// safe to retry on connection drop.
func (c *Client) GetJob(ctx context.Context, jobID int64) (*jobInfo, error) {
	result, err := c.Call(ctx, "core.get_jobs",
		[]interface{}{[]interface{}{[]interface{}{"id", "=", jobID}}},
		CallOptions{Idempotent: true, Read: true})
	if err != nil {
		return nil, err
	}
	var jobs []jobInfo
	if err := json.Unmarshal(result, &jobs); err != nil {
		return nil, fmt.Errorf("GetJob %d: decode result: %w", jobID, err)
	}
	if len(jobs) == 0 {
		return nil, fmt.Errorf("GetJob %d: not found", jobID)
	}
	return &jobs[0], nil
}
