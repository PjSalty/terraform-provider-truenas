package wsclient

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestIsSensitiveKey(t *testing.T) {
	cases := map[string]bool{
		"password":           true,
		"Password":           true,
		"bind_password":      true,
		"my_secret":          true,
		"api_key":            true,
		"APIKEY":             true,
		"x-auth-token":       true,
		"private_key":        true,
		"Authorization":      true,
		"username":           false,
		"hostname":           false,
		"id":                 false,
		"":                   false,
		"refresh_token":      true,
		"client_secret":      true,
		"aws_session_token":  true,
		"non_sensitive_name": false,
	}
	for k, want := range cases {
		t.Run(k, func(t *testing.T) {
			if got := isSensitiveKey(k); got != want {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", k, got, want)
			}
		})
	}
}

func TestRedactJSONBody(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		out := redactJSONBody(nil)
		if len(out) != 0 {
			t.Errorf("empty input should return empty, got %q", string(out))
		}
	})

	t.Run("non-JSON short body", func(t *testing.T) {
		in := []byte("not json here")
		out := redactJSONBody(in)
		if !bytes.Equal(out, in) {
			t.Errorf("short non-JSON should pass through, got %q", string(out))
		}
	})

	t.Run("non-JSON long body truncates", func(t *testing.T) {
		long := bytes.Repeat([]byte("X"), 1024)
		out := redactJSONBody(long)
		if !strings.Contains(string(out), "non-JSON body") {
			t.Errorf("expected truncation marker in %q", string(out))
		}
		if len(out) > 600 { // 512 cap + marker
			t.Errorf("truncated body too long: %d", len(out))
		}
	})

	t.Run("redact at root", func(t *testing.T) {
		in := []byte(`{"username":"alice","password":"hunter2"}`)
		out := redactJSONBody(in)
		var obj map[string]interface{}
		if err := json.Unmarshal(out, &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if obj["password"] != redactedPlaceholder {
			t.Errorf("password not redacted: %v", obj["password"])
		}
		if obj["username"] != "alice" {
			t.Errorf("username altered: %v", obj["username"])
		}
	})

	t.Run("redact nested", func(t *testing.T) {
		in := []byte(`{"config":{"api_key":"abc123","port":443}}`)
		out := redactJSONBody(in)
		if !strings.Contains(string(out), redactedPlaceholder) {
			t.Errorf("nested api_key not redacted: %q", string(out))
		}
	})

	t.Run("redact in array", func(t *testing.T) {
		in := []byte(`{"users":[{"name":"alice","password":"x"},{"name":"bob","password":"y"}]}`)
		out := redactJSONBody(in)
		count := strings.Count(string(out), redactedPlaceholder)
		if count != 2 {
			t.Errorf("expected 2 redactions, got %d in %q", count, string(out))
		}
	})

	t.Run("scalars pass through", func(t *testing.T) {
		in := []byte(`{"x":42,"y":true,"z":null}`)
		out := redactJSONBody(in)
		var obj map[string]interface{}
		_ = json.Unmarshal(out, &obj)
		if obj["x"].(float64) != 42 || obj["y"].(bool) != true || obj["z"] != nil {
			t.Errorf("scalars altered: %v", obj)
		}
	})
}

func TestRedactMessage(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"plain message", "plain message"},
		{"value of password is wrong", "value of " + redactedPlaceholder},
		{"PASSWORD=hunter2", redactedPlaceholder},
		{"api_key 'abc' not allowed", redactedPlaceholder},
		{"unrelated error here", "unrelated error here"},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			got := redactMessage(tt.in)
			if got != tt.want {
				t.Errorf("redactMessage(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
