package resources

import (
	"encoding/json"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestParseQuotaValue exercises quota-string parsing which is used by both
// dataset and zvol resources when mapping the API response.
func TestParseQuotaValue(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    int64
		wantErr bool
	}{
		{name: "empty", in: "", want: 0},
		{name: "zero", in: "0", want: 0},
		{name: "1 GiB", in: "1073741824", want: 1073741824},
		{name: "100 bytes", in: "100", want: 100},
		{name: "max int64", in: "9223372036854775807", want: 9223372036854775807},
		{name: "negative", in: "-1", want: -1},
		{name: "invalid number", in: "not-a-number", wantErr: true},
		{name: "float rejected", in: "1.5", wantErr: true},
		{name: "hex rejected", in: "0x10", wantErr: true},
		{name: "trailing space", in: "100 ", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseQuotaValue(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseQuotaValue(%q) = %d, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseQuotaValue(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("parseQuotaValue(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestExtractCredentialsID verifies that extractCredentialsID handles both
// the integer shape and the expanded-object shape the TrueNAS API uses
// interchangeably for the `credentials` field on cloud_sync/cloud_backup.
func TestExtractCredentialsID(t *testing.T) {
	cases := []struct {
		name string
		raw  json.RawMessage
		want int
	}{
		{name: "empty raw", raw: nil, want: 0},
		{name: "zero length", raw: json.RawMessage(""), want: 0},
		{name: "plain int", raw: json.RawMessage("42"), want: 42},
		{name: "plain zero", raw: json.RawMessage("0"), want: 0},
		{name: "large int", raw: json.RawMessage("2147483647"), want: 2147483647},
		{name: "object with id", raw: json.RawMessage(`{"id":7,"name":"rclone","provider":"s3"}`), want: 7},
		{name: "object with zero id", raw: json.RawMessage(`{"id":0}`), want: 0},
		{name: "object with string id and no numeric id", raw: json.RawMessage(`{"name":"x"}`), want: 0},
		{name: "array fallback to zero", raw: json.RawMessage("[1,2,3]"), want: 0},
		{name: "null", raw: json.RawMessage("null"), want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCredentialsID(tc.raw)
			if got != tc.want {
				t.Errorf("extractCredentialsID(%s) = %d, want %d", string(tc.raw), got, tc.want)
			}
		})
	}
}

// TestDatasetResponse_GetComments exercises the cross-version compatibility
// shim: 25.10 places comments in user_properties while 25.04 places them
// at the top-level Comments field.
func TestDatasetResponse_GetComments(t *testing.T) {
	cases := []struct {
		name string
		ds   *client.DatasetResponse
		want string
	}{
		{
			name: "25.04 top-level comments",
			ds: &client.DatasetResponse{
				Comments: &client.PropertyValue{Value: "legacy note"},
			},
			want: "legacy note",
		},
		{
			name: "25.10 user_properties comments",
			ds: &client.DatasetResponse{
				UserProperties: map[string]*client.PropertyValue{
					"comments": {Value: "new shape note"},
				},
			},
			want: "new shape note",
		},
		{
			name: "25.10 preferred when both present",
			ds: &client.DatasetResponse{
				Comments: &client.PropertyValue{Value: "old"},
				UserProperties: map[string]*client.PropertyValue{
					"comments": {Value: "new"},
				},
			},
			want: "new",
		},
		{
			name: "25.10 empty value falls back to 25.04",
			ds: &client.DatasetResponse{
				Comments: &client.PropertyValue{Value: "fallback"},
				UserProperties: map[string]*client.PropertyValue{
					"comments": {Value: ""},
				},
			},
			want: "fallback",
		},
		{
			name: "neither returns empty",
			ds:   &client.DatasetResponse{},
			want: "",
		},
		{
			name: "nil user_properties entry ignored",
			ds: &client.DatasetResponse{
				UserProperties: map[string]*client.PropertyValue{"comments": nil},
				Comments:       &client.PropertyValue{Value: "fallback"},
			},
			want: "fallback",
		},
		{
			name: "unrelated user_properties key ignored",
			ds: &client.DatasetResponse{
				UserProperties: map[string]*client.PropertyValue{"other": {Value: "x"}},
				Comments:       &client.PropertyValue{Value: "back"},
			},
			want: "back",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ds.GetComments()
			if got != tc.want {
				t.Errorf("GetComments() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDatasetResponse_GetVolsize verifies zvol size parsing edge cases.
func TestDatasetResponse_GetVolsize(t *testing.T) {
	cases := []struct {
		name string
		ds   *client.DatasetResponse
		want int64
	}{
		{name: "nil volsize returns 0", ds: &client.DatasetResponse{}, want: 0},
		{name: "empty rawvalue returns 0", ds: &client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: ""}}, want: 0},
		{name: "16 MiB", ds: &client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: "16777216"}}, want: 16777216},
		{name: "1 GiB", ds: &client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: "1073741824"}}, want: 1073741824},
		{name: "zero string", ds: &client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: "0"}}, want: 0},
		{name: "invalid returns 0", ds: &client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: "not-a-number"}}, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ds.GetVolsize()
			if got != tc.want {
				t.Errorf("GetVolsize() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestDatasetResponse_GetVolblocksize verifies block-size accessor.
func TestDatasetResponse_GetVolblocksize(t *testing.T) {
	cases := []struct {
		name string
		ds   *client.DatasetResponse
		want string
	}{
		{name: "nil returns empty", ds: &client.DatasetResponse{}, want: ""},
		{name: "16K", ds: &client.DatasetResponse{Volblocksize: &client.PropertyValue{Value: "16K"}}, want: "16K"},
		{name: "128K", ds: &client.DatasetResponse{Volblocksize: &client.PropertyValue{Value: "128K"}}, want: "128K"},
		{name: "4K", ds: &client.DatasetResponse{Volblocksize: &client.PropertyValue{Value: "4K"}}, want: "4K"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ds.GetVolblocksize()
			if got != tc.want {
				t.Errorf("GetVolblocksize() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestISCSIExtent_GetDisk exercises the json.RawMessage disk accessor which
// must handle null, missing, and string-encoded values.
func TestISCSIExtent_GetDisk(t *testing.T) {
	cases := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "nil raw", raw: nil, want: ""},
		{name: "zero length", raw: json.RawMessage(""), want: ""},
		{name: "null", raw: json.RawMessage("null"), want: ""},
		{name: "simple string", raw: json.RawMessage(`"zvol/tank/vol1"`), want: "zvol/tank/vol1"},
		{name: "empty string", raw: json.RawMessage(`""`), want: ""},
		{name: "non-string value", raw: json.RawMessage("123"), want: ""},
		{name: "array non-string", raw: json.RawMessage("[1,2]"), want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &client.ISCSIExtent{Disk: tc.raw}
			got := e.GetDisk()
			if got != tc.want {
				t.Errorf("GetDisk(%s) = %q, want %q", string(tc.raw), got, tc.want)
			}
		})
	}
}

// TestISCSIExtent_GetFilesize exercises the json.RawMessage filesize
// accessor which must handle both numeric and string-encoded shapes.
func TestISCSIExtent_GetFilesize(t *testing.T) {
	cases := []struct {
		name string
		raw  json.RawMessage
		want int64
	}{
		{name: "nil raw", raw: nil, want: 0},
		{name: "null", raw: json.RawMessage("null"), want: 0},
		{name: "integer number", raw: json.RawMessage("1073741824"), want: 1073741824},
		{name: "zero number", raw: json.RawMessage("0"), want: 0},
		{name: "large number", raw: json.RawMessage("9223372036854775807"), want: 9223372036854775807},
		{name: "string number", raw: json.RawMessage(`"536870912"`), want: 536870912},
		{name: "string zero", raw: json.RawMessage(`"0"`), want: 0},
		{name: "invalid string", raw: json.RawMessage(`"not-a-number"`), want: 0},
		{name: "array rejected", raw: json.RawMessage("[1]"), want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &client.ISCSIExtent{Filesize: tc.raw}
			got := e.GetFilesize()
			if got != tc.want {
				t.Errorf("GetFilesize(%s) = %d, want %d", string(tc.raw), got, tc.want)
			}
		})
	}
}
