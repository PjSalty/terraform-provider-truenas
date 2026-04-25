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

func TestGetMailConfig_Success(t *testing.T) {
	ctx := context.Background()
	user := "admin"
	want := client.MailConfig{
		ID: 1, FromEmail: "from@example.com", FromName: "Admin",
		OutgoingServer: "smtp.example.com", Port: 587, Security: "TLS",
		SMTP: true, User: &user, Pass: "secret",
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/mail") {
			t.Errorf("path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, want)
	}))
	got, err := c.GetMailConfig(ctx)
	if err != nil {
		t.Fatalf("GetMailConfig: %v", err)
	}
	if got.FromEmail != "from@example.com" {
		t.Errorf("FromEmail: %q", got.FromEmail)
	}
	if got.Port != 587 {
		t.Errorf("Port: %d", got.Port)
	}
	if got.User == nil || *got.User != "admin" {
		t.Errorf("User wrong")
	}
}

func TestGetMailConfig_NotFound(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
	}))
	_, err := c.GetMailConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("IsNotFound: %v", err)
	}
}

func TestGetMailConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "xxx")
	}))
	_, err := c.GetMailConfig(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing mail config") {
		t.Errorf("got: %v", err)
	}
}

func TestGetMailConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "boom"})
	}))
	_, err := c.GetMailConfig(ctx)
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

func TestUpdateMailConfig_Success(t *testing.T) {
	ctx := context.Background()
	fromEmail := "new@example.com"
	fromName := "System"
	server := "smtp.new.com"
	port := 465
	security := "SSL"
	smtp := true
	user := "user"
	pass := "pass"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req client.MailConfigUpdateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.FromEmail == nil || *req.FromEmail != "new@example.com" {
			t.Errorf("FromEmail wrong")
		}
		if req.Port == nil || *req.Port != 465 {
			t.Errorf("Port wrong")
		}
		if req.Security == nil || *req.Security != "SSL" {
			t.Errorf("Security wrong")
		}
		writeJSON(w, http.StatusOK, client.MailConfig{
			ID: 1, FromEmail: "new@example.com", FromName: "System",
			OutgoingServer: "smtp.new.com", Port: 465, Security: "SSL",
			SMTP: true, User: &user, Pass: "pass",
		})
	}))

	got, err := c.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{
		FromEmail: &fromEmail, FromName: &fromName, OutgoingServer: &server,
		Port: &port, Security: &security, SMTP: &smtp, User: &user, Pass: &pass,
	})
	if err != nil {
		t.Fatalf("UpdateMailConfig: %v", err)
	}
	if got.FromEmail != "new@example.com" {
		t.Errorf("FromEmail: %q", got.FromEmail)
	}
}

func TestUpdateMailConfig_OmitEmpty(t *testing.T) {
	ctx := context.Background()
	port := 25
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "port") {
			t.Errorf("missing port: %s", body)
		}
		if strings.Contains(string(body), "fromemail") {
			t.Errorf("should omit fromemail: %s", body)
		}
		writeJSON(w, http.StatusOK, client.MailConfig{ID: 1, Port: 25})
	}))
	_, err := c.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{Port: &port})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestUpdateMailConfig_ServerError(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad port"})
	}))
	_, err := c.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("type: %T", err)
	}
	if apiErr.Message != "bad port" {
		t.Errorf("message: %q", apiErr.Message)
	}
}

func TestUpdateMailConfig_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "nope")
	}))
	_, err := c.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing mail config update") {
		t.Errorf("got: %v", err)
	}
}

func TestUpdateMailConfig_404(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
	}))
	_, err := c.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound: %v", err)
	}
}

func TestGetMailConfig_NilUser(t *testing.T) {
	ctx := context.Background()
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"fromemail":"","fromname":"","outgoingserver":"","port":25,"security":"","smtp":false,"user":null,"pass":""}`))
	}))
	got, err := c.GetMailConfig(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.User != nil {
		t.Errorf("User should be nil: %v", *got.User)
	}
}

func TestGetMailConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var method string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		writeJSON(w, http.StatusOK, client.MailConfig{ID: 1})
	}))
	if _, err := c.GetMailConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet {
		t.Errorf("method: %s", method)
	}
}

func TestUpdateMailConfig_MethodCheck(t *testing.T) {
	ctx := context.Background()
	var method string
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		writeJSON(w, http.StatusOK, client.MailConfig{ID: 1})
	}))
	if _, err := c.UpdateMailConfig(ctx, &client.MailConfigUpdateRequest{}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut {
		t.Errorf("method: %s", method)
	}
}
