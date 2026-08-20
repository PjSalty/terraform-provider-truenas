package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// The update path needs three distinguishable wire states, because
// middleware's __set_password branches on them differently:
//
//	key absent    -> returns early, stored hash untouched
//	key null      -> falsy, sets unixhash='*' (the account becomes passwordless)
//	key "value"   -> hashes it
//
// A plain *string collapses the first two, which is why Password is a
// json.RawMessage. These pin all three.

func TestUserUpdateRequest_passwordOmittedWhenUnset(t *testing.T) {
	b, err := json.Marshal(&UserUpdateRequest{FullName: "svc"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "password") {
		t.Errorf("password key present on a request that never set it: %s", b)
	}
}

func TestUserUpdateRequest_clearPasswordEmitsExplicitNull(t *testing.T) {
	r := &UserUpdateRequest{FullName: "svc"}
	r.ClearPassword()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"password":null`) {
		t.Errorf("expected an explicit null so middleware wipes the hash, got: %s", b)
	}
}

func TestUserUpdateRequest_setPasswordEmitsValue(t *testing.T) {
	r := &UserUpdateRequest{}
	r.SetPassword("hunter2")
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"password":"hunter2"`) {
		t.Errorf("password value not encoded: %s", b)
	}
}

// A value needing JSON escaping must survive intact; SetPassword marshals
// rather than concatenating, so a quote cannot break out of the string.
func TestUserUpdateRequest_setPasswordEscapes(t *testing.T) {
	r := &UserUpdateRequest{}
	r.SetPassword(`a"b\c`)
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("the encoded body is not valid JSON: %s", b)
	}
	if round.Password != `a"b\c` {
		t.Errorf("round trip changed the password: %q", round.Password)
	}
}

// Create only needs two states, so a pointer is enough there, but the
// omission has to be real: "" is rejected by middleware's NonEmptyString.
func TestUserCreateRequest_passwordOmittedWhenNil(t *testing.T) {
	pd := true
	b, err := json.Marshal(&UserCreateRequest{Username: "svc", PasswordDisabled: &pd})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), `"password"`) {
		t.Errorf("password key present on a passwordless create: %s", b)
	}
	if !strings.Contains(string(b), `"password_disabled":true`) {
		t.Errorf("password_disabled not sent: %s", b)
	}
}

func TestUserCreateRequest_passwordSentWhenSet(t *testing.T) {
	pw := "hunter2"
	b, err := json.Marshal(&UserCreateRequest{Username: "svc", Password: &pw})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"password":"hunter2"`) {
		t.Errorf("password not sent: %s", b)
	}
}
