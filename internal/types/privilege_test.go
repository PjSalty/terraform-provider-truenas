package types

import (
	"reflect"
	"testing"
)

func TestPrivilege_LocalGroupGIDs_empty(t *testing.T) {
	p := &Privilege{}
	got := p.LocalGroupGIDs()
	if len(got) != 0 {
		t.Errorf("empty: got %v", got)
	}
}

func TestPrivilege_LocalGroupGIDs_multiple(t *testing.T) {
	p := &Privilege{
		LocalGroups: []PrivilegeGroup{
			{ID: 1, GID: 1000, Name: "admins"},
			{ID: 2, GID: 1001, Name: "operators"},
		},
	}
	got := p.LocalGroupGIDs()
	want := []int{1000, 1001}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPrivilege_DSGroupStrings_empty(t *testing.T) {
	p := &Privilege{}
	got := p.DSGroupStrings()
	if len(got) != 0 {
		t.Errorf("empty: got %v", got)
	}
}

func TestPrivilege_DSGroupStrings_string(t *testing.T) {
	// SIDs come back as strings.
	p := &Privilege{DSGroups: []interface{}{"S-1-5-21-12345-67890"}}
	got := p.DSGroupStrings()
	want := []string{"S-1-5-21-12345-67890"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPrivilege_DSGroupStrings_float64(t *testing.T) {
	// GIDs decode as float64 from json.Unmarshal into interface{}.
	p := &Privilege{DSGroups: []interface{}{float64(2000)}}
	got := p.DSGroupStrings()
	want := []string{"2000"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPrivilege_DSGroupStrings_int(t *testing.T) {
	// In-process construction (not via JSON) may use int.
	p := &Privilege{DSGroups: []interface{}{int(3000)}}
	got := p.DSGroupStrings()
	want := []string{"3000"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPrivilege_DSGroupStrings_mixed(t *testing.T) {
	p := &Privilege{DSGroups: []interface{}{
		"S-1-5-21-1",
		float64(1500),
		int(2500),
	}}
	got := p.DSGroupStrings()
	want := []string{"S-1-5-21-1", "1500", "2500"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPrivilege_DSGroupStrings_unsupportedTypeIgnored(t *testing.T) {
	// Unsupported types (bool, nil, slice, map) silently drop.
	p := &Privilege{DSGroups: []interface{}{
		true,
		nil,
		[]string{"nope"},
		map[string]int{"foo": 1},
		"keep-me",
	}}
	got := p.DSGroupStrings()
	want := []string{"keep-me"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
