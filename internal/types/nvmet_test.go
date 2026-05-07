package types

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// NVMetPort.GetAddrTrsvcid
// =============================================================================

func TestNVMetPort_GetAddrTrsvcid_unset(t *testing.T) {
	p := &NVMetPort{}
	if got := p.GetAddrTrsvcid(); got != 0 {
		t.Errorf("unset: got %d", got)
	}
}

func TestNVMetPort_GetAddrTrsvcid_null(t *testing.T) {
	p := &NVMetPort{AddrTrsvcid: json.RawMessage(`null`)}
	if got := p.GetAddrTrsvcid(); got != 0 {
		t.Errorf("null: got %d", got)
	}
}

func TestNVMetPort_GetAddrTrsvcid_int(t *testing.T) {
	p := &NVMetPort{AddrTrsvcid: json.RawMessage(`4420`)}
	if got := p.GetAddrTrsvcid(); got != 4420 {
		t.Errorf("int: got %d", got)
	}
}

func TestNVMetPort_GetAddrTrsvcid_stringNumeric(t *testing.T) {
	p := &NVMetPort{AddrTrsvcid: json.RawMessage(`"4420"`)}
	if got := p.GetAddrTrsvcid(); got != 4420 {
		t.Errorf("string numeric: got %d", got)
	}
}

func TestNVMetPort_GetAddrTrsvcid_stringNonNumeric(t *testing.T) {
	p := &NVMetPort{AddrTrsvcid: json.RawMessage(`"not-a-port"`)}
	if got := p.GetAddrTrsvcid(); got != 0 {
		t.Errorf("string non-numeric: got %d", got)
	}
}

func TestNVMetPort_GetAddrTrsvcid_object(t *testing.T) {
	p := &NVMetPort{AddrTrsvcid: json.RawMessage(`{"foo":"bar"}`)}
	if got := p.GetAddrTrsvcid(); got != 0 {
		t.Errorf("object: got %d", got)
	}
}

// =============================================================================
// NVMetNamespace.EffectiveSubsysID
// =============================================================================

func TestNVMetNamespace_EffectiveSubsysID_nested(t *testing.T) {
	ns := &NVMetNamespace{Subsys: &NVMetNamespaceSubsys{ID: 7}, SubsysID: 99}
	if got := ns.EffectiveSubsysID(); got != 7 {
		t.Errorf("nested wins: got %d", got)
	}
}

func TestNVMetNamespace_EffectiveSubsysID_flat(t *testing.T) {
	ns := &NVMetNamespace{SubsysID: 99}
	if got := ns.EffectiveSubsysID(); got != 99 {
		t.Errorf("flat: got %d", got)
	}
}

func TestNVMetNamespace_EffectiveSubsysID_nestedZero(t *testing.T) {
	// Nested ID == 0 falls through to flat.
	ns := &NVMetNamespace{Subsys: &NVMetNamespaceSubsys{ID: 0}, SubsysID: 42}
	if got := ns.EffectiveSubsysID(); got != 42 {
		t.Errorf("nested-zero falls through: got %d", got)
	}
}

func TestNVMetNamespace_EffectiveSubsysID_empty(t *testing.T) {
	ns := &NVMetNamespace{}
	if got := ns.EffectiveSubsysID(); got != 0 {
		t.Errorf("empty: got %d", got)
	}
}

// =============================================================================
// NVMetHostSubsys.EffectiveHostID / EffectiveSubsysID
// =============================================================================

func TestNVMetHostSubsys_EffectiveHostID_nested(t *testing.T) {
	hs := &NVMetHostSubsys{Host: &NVMetHostSubsysHost{ID: 3}, HostID: 88}
	if got := hs.EffectiveHostID(); got != 3 {
		t.Errorf("nested wins: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveHostID_flat(t *testing.T) {
	hs := &NVMetHostSubsys{HostID: 88}
	if got := hs.EffectiveHostID(); got != 88 {
		t.Errorf("flat: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveHostID_nestedZero(t *testing.T) {
	hs := &NVMetHostSubsys{Host: &NVMetHostSubsysHost{ID: 0}, HostID: 5}
	if got := hs.EffectiveHostID(); got != 5 {
		t.Errorf("nested-zero falls through: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveHostID_empty(t *testing.T) {
	hs := &NVMetHostSubsys{}
	if got := hs.EffectiveHostID(); got != 0 {
		t.Errorf("empty: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveSubsysID_nested(t *testing.T) {
	hs := &NVMetHostSubsys{Subsys: &NVMetHostSubsysSubsys{ID: 4}, SubsysID: 77}
	if got := hs.EffectiveSubsysID(); got != 4 {
		t.Errorf("nested wins: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveSubsysID_flat(t *testing.T) {
	hs := &NVMetHostSubsys{SubsysID: 77}
	if got := hs.EffectiveSubsysID(); got != 77 {
		t.Errorf("flat: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveSubsysID_nestedZero(t *testing.T) {
	hs := &NVMetHostSubsys{Subsys: &NVMetHostSubsysSubsys{ID: 0}, SubsysID: 9}
	if got := hs.EffectiveSubsysID(); got != 9 {
		t.Errorf("nested-zero falls through: got %d", got)
	}
}

func TestNVMetHostSubsys_EffectiveSubsysID_empty(t *testing.T) {
	hs := &NVMetHostSubsys{}
	if got := hs.EffectiveSubsysID(); got != 0 {
		t.Errorf("empty: got %d", got)
	}
}

// =============================================================================
// NVMetPortSubsys.EffectivePortID / EffectiveSubsysID
// =============================================================================

func TestNVMetPortSubsys_EffectivePortID_nested(t *testing.T) {
	ps := &NVMetPortSubsys{Port: &NVMetPortSubsysPort{ID: 11}, PortID: 99}
	if got := ps.EffectivePortID(); got != 11 {
		t.Errorf("nested wins: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectivePortID_flat(t *testing.T) {
	ps := &NVMetPortSubsys{PortID: 22}
	if got := ps.EffectivePortID(); got != 22 {
		t.Errorf("flat: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectivePortID_nestedZero(t *testing.T) {
	ps := &NVMetPortSubsys{Port: &NVMetPortSubsysPort{ID: 0}, PortID: 33}
	if got := ps.EffectivePortID(); got != 33 {
		t.Errorf("nested-zero falls through: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectivePortID_empty(t *testing.T) {
	ps := &NVMetPortSubsys{}
	if got := ps.EffectivePortID(); got != 0 {
		t.Errorf("empty: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectiveSubsysID_nested(t *testing.T) {
	ps := &NVMetPortSubsys{Subsys: &NVMetPortSubsysSubsys{ID: 13}, SubsysID: 99}
	if got := ps.EffectiveSubsysID(); got != 13 {
		t.Errorf("nested wins: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectiveSubsysID_flat(t *testing.T) {
	ps := &NVMetPortSubsys{SubsysID: 44}
	if got := ps.EffectiveSubsysID(); got != 44 {
		t.Errorf("flat: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectiveSubsysID_nestedZero(t *testing.T) {
	ps := &NVMetPortSubsys{Subsys: &NVMetPortSubsysSubsys{ID: 0}, SubsysID: 55}
	if got := ps.EffectiveSubsysID(); got != 55 {
		t.Errorf("nested-zero falls through: got %d", got)
	}
}

func TestNVMetPortSubsys_EffectiveSubsysID_empty(t *testing.T) {
	ps := &NVMetPortSubsys{}
	if got := ps.EffectiveSubsysID(); got != 0 {
		t.Errorf("empty: got %d", got)
	}
}
