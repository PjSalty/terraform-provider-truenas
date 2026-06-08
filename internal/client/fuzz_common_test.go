package client_test

import "testing"

// commonSeeds mirrors the corpus used by internal/types — a mix of
// well-formed JSON, edge JSON, and malformed bytes so go test -fuzz
// has a sensible starting point. The fuzz engine mutates from there.
var commonSeeds = [][]byte{
	[]byte(`{}`),
	[]byte(`null`),
	[]byte(`[]`),
	[]byte(`""`),
	[]byte(`0`),
	[]byte(`false`),
	[]byte(`{"id":1}`),
	[]byte(`{"id":-1}`),
	[]byte(`{"id":99999999999999999999}`),
	[]byte(`{"id":1.7e308}`),
	[]byte(`{"name":""}`),
	[]byte(`{"name":" "}`),
	[]byte(`{"path":"/"}`),
	[]byte(`{"enabled":true}`),
	[]byte(`{"enabled":"yes"}`),
	[]byte(`{"unknown_field":42,"another":[1,2,3]}`),
	[]byte(`{"id":1,"nested":{"deep":{"deeper":{"x":1}}}}`),
	[]byte(`{"id":null,"name":null}`),
	[]byte(`{`),
	[]byte(`{"id":}`),
	[]byte(`{"id":1,`),
	{0x00, 0x01, 0x02},
	[]byte(`{"id":1,"name":"trailing"}`),
}

func addCommonSeeds(f *testing.F) {
	for _, s := range commonSeeds {
		f.Add(s)
	}
}
