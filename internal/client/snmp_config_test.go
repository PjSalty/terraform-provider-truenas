package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestGetSNMPConfig_Success(t *testing.T) {
	ctx := context.Background()
	priv := "AES"
	pass := "secret"
	want := client.SNMPConfig{
		ID: 1, Community: "public", Contact: "admin", Location: "dc1",
		V3: true, V3Username: "snmpuser", V3AuthType: "SHA", V3Password: "pw",
		V3PrivProto: &priv, V3PrivPassphrase: &pass,
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/snmp") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetSNMPConfig(ctx)
	if err != nil {
		t.Fatalf("GetSNMPConfig: %v", err)
	}
	if got.Community != "public" {
		t.Errorf("Community: %q", got.Community)
	}
	if !got.V3 {
		t.Errorf("V3 expected true")
	}
	if got.V3PrivProto == nil || *got.V3PrivProto != "AES" {
		t.Errorf("V3PrivProto wrong")
	}
}

func TestGetSNMPConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetSNMPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetSNMPConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.GetSNMPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing SNMP config") {
		t.Errorf("got: %v", err)
	}
}

func TestGetSNMPConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "boom"})
	}))
	_, err := c.GetSNMPConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status: %d", apiErr.StatusCode)
	}
}

func TestUpdateSNMPConfig_Success(t *testing.T) {
	ctx := context.Background()
	community := "private"
	v3 := true
	user := "admin"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.SNMPConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Community == nil || *req.Community != "private" {
			t.Errorf("Community wrong")
		}
		if req.V3 == nil || !*req.V3 {
			t.Errorf("V3 wrong")
		}
		if req.V3Username == nil || *req.V3Username != "admin" {
			t.Errorf("V3Username wrong")
		}
		writeJSON(w, http.StatusOK, client.SNMPConfig{ID: 1, Community: "private", V3: true, V3Username: "admin"})
	}))

	got, err := c.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{
		Community: &community, V3: &v3, V3Username: &user,
	})
	if err != nil {
		t.Fatalf("UpdateSNMPConfig: %v", err)
	}
	if got.Community != "private" {
		t.Errorf("Community: %q", got.Community)
	}
}

func TestUpdateSNMPConfig_OmitEmpty(t *testing.T) {
	ctx := context.Background()
	community := "public"
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "community") {
			t.Errorf("missing community: %s", body)
		}
		if strings.Contains(string(body), "v3_authtype") {
			t.Errorf("should omit v3_authtype: %s", body)
		}
		writeJSON(w, http.StatusOK, client.SNMPConfig{ID: 1, Community: "public"})
	}))
	_, err := c.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{Community: &community})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestUpdateSNMPConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
	}))
	_, err := c.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "bad" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestUpdateSNMPConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-json")
	}))
	_, err := c.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing SNMP config update") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateSNMPConfig_404(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
	}))
	_, err := c.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetSNMPConfig_Defaults(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, client.SNMPConfig{ID: 2})
	}))
	got, err := c.GetSNMPConfig(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != 2 || got.Community != "" {
		t.Errorf("defaults: %+v", got)
	}
}

func TestUpdateSNMPConfig_AllFields(t *testing.T) {
	ctx := context.Background()
	community := "c"
	contact := "ct"
	loc := "l"
	v3 := true
	user := "u"
	auth := "SHA"
	pass := "p"
	priv := "AES"
	ppass := "pp"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		for _, key := range []string{"community", "contact", "location", "v3",
			"v3_username", "v3_authtype", "v3_password", "v3_privproto", "v3_privpassphrase"} {
			if !strings.Contains(string(body), key) {
				t.Errorf("missing key %q", key)
			}
		}
		writeJSON(w, http.StatusOK, client.SNMPConfig{ID: 1})
	}))
	_, err := c.UpdateSNMPConfig(ctx, &client.SNMPConfigUpdateRequest{
		Community: &community, Contact: &contact, Location: &loc, V3: &v3,
		V3Username: &user, V3AuthType: &auth, V3Password: &pass,
		V3PrivProto: &priv, V3PrivPassphrase: &ppass,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestGetSNMPConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var method string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		writeJSON(w, http.StatusOK, client.SNMPConfig{ID: 1})
	}))
	if _, err := c.GetSNMPConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet {
		t.Errorf("method: %s", method)
	}
}
