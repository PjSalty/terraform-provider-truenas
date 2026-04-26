package client_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestIsNotFound covers every IsNotFound code path, including the
// regression for wrapped APIErrors (errors.As must unwrap them).
func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "raw 404 APIError",
			err: &client.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "not found",
			},
			want: true,
		},
		{
			name: "wrapped 404 APIError via fmt.Errorf %w",
			// This was a regression — domain methods wrap errors with
			// fmt.Errorf("...: %w", err), so IsNotFound must call
			// errors.As not a direct type assertion.
			err: fmt.Errorf("getting dataset %q: %w", "tank/x", &client.APIError{
				StatusCode: http.StatusNotFound,
			}),
			want: true,
		},
		{
			name: "double-wrapped 404 APIError",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("inner: %w", &client.APIError{
					StatusCode: http.StatusNotFound,
				})),
			want: true,
		},
		{
			name: "422 with 'does not exist' body",
			err: &client.APIError{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       `{"message": "Object with id=5 does not exist"}`,
			},
			want: true,
		},
		{
			name: "wrapped 422 with 'does not exist'",
			err: fmt.Errorf("deleting NFS share %d: %w", 99, &client.APIError{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       `[{"message": "does not exist"}]`,
			}),
			want: true,
		},
		{
			name: "422 with unrelated message",
			err: &client.APIError{
				StatusCode: http.StatusUnprocessableEntity,
				Body:       `{"message": "Extra inputs are not permitted"}`,
			},
			want: false,
		},
		{
			name: "500 server error",
			err: &client.APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "boom",
			},
			want: false,
		},
		{
			name: "400 bad request",
			err: &client.APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "bad",
			},
			want: false,
		},
		{
			name: "random non-APIError",
			err:  errors.New("some random error"),
			want: false,
		},
		{
			name: "wrapped non-APIError",
			err:  fmt.Errorf("wrap: %w", errors.New("plain")),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := client.IsNotFound(tc.err)
			if got != tc.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
