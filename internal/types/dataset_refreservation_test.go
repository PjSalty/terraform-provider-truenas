package types

import "testing"

func TestDatasetResponse_GetRefreservation(t *testing.T) {
	cases := []struct {
		name   string
		prop   *PropertyRawVal
		want   int64
		wantOK bool
	}{
		{"absent", nil, 0, false},
		{"empty rawvalue", &PropertyRawVal{Value: "none"}, 0, false},
		{"unparsable rawvalue", &PropertyRawVal{Rawvalue: "abc"}, 0, false},
		{"none is 0", &PropertyRawVal{Value: "none", Rawvalue: "0"}, 0, true},
		{"thick reservation in bytes", &PropertyRawVal{Value: "17M", Rawvalue: "17825792"}, 17825792, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := (&DatasetResponse{Refreservation: tc.prop}).GetRefreservation()
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("GetRefreservation() = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
