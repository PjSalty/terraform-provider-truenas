package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// =============================================================================
// Realm
// =============================================================================

func TestListKerberosRealms(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.realm.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{"id": 1, "realm": "EXAMPLE.COM",
				"kdc": []interface{}{"kdc.example.com"}, "admin_server": []interface{}{}, "kpasswd_server": []interface{}{}},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	realms, err := c.ListKerberosRealms(ctx)
	if err != nil {
		t.Fatalf("ListKerberosRealms: %v", err)
	}
	if len(realms) != 1 || realms[0].Realm != "EXAMPLE.COM" {
		t.Errorf("got %+v", realms)
	}
}

func TestListKerberosRealms_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListKerberosRealms(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing kerberos realms") {
		t.Errorf("got %v", err)
	}
}

func TestListKerberosRealms_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListKerberosRealms(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetKerberosRealm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.realm.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "realm": "MY.REALM"}, nil
	})
	c, _ := ts.NewClient(ctx)
	realm, err := c.GetKerberosRealm(ctx, 5)
	if err != nil {
		t.Fatalf("GetKerberosRealm: %v", err)
	}
	if realm.Realm != "MY.REALM" {
		t.Errorf("got %+v", realm)
	}
}

func TestGetKerberosRealm_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKerberosRealm(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting kerberos realm") {
		t.Errorf("got %v", err)
	}
}

func TestGetKerberosRealm_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKerberosRealm(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestCreateKerberosRealm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.realm.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "realm": "NEW.REALM"}, nil
	})
	c, _ := ts.NewClient(ctx)
	realm, err := c.CreateKerberosRealm(ctx, &types.KerberosRealmCreateRequest{
		Realm: "NEW.REALM", KDC: []string{"kdc1"},
	})
	if err != nil {
		t.Fatalf("CreateKerberosRealm: %v", err)
	}
	if realm.ID != 9 {
		t.Errorf("got %+v", realm)
	}
}

func TestCreateKerberosRealm_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateKerberosRealm(ctx, &types.KerberosRealmCreateRequest{Realm: "X"})
	if err == nil || !strings.Contains(err.Error(), "creating kerberos realm") {
		t.Errorf("got %v", err)
	}
}

func TestCreateKerberosRealm_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateKerberosRealm(ctx, &types.KerberosRealmCreateRequest{Realm: "X"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateKerberosRealm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.realm.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "realm": "RENAMED.REALM"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r := "RENAMED.REALM"
	realm, err := c.UpdateKerberosRealm(ctx, 9, &types.KerberosRealmUpdateRequest{Realm: &r})
	if err != nil {
		t.Fatalf("UpdateKerberosRealm: %v", err)
	}
	if realm.Realm != "RENAMED.REALM" {
		t.Errorf("got %+v", realm)
	}
}

func TestUpdateKerberosRealm_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKerberosRealm(ctx, 9, &types.KerberosRealmUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating kerberos realm") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateKerberosRealm_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKerberosRealm(ctx, 9, &types.KerberosRealmUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteKerberosRealm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.realm.delete" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteKerberosRealm(ctx, 9); err != nil {
		t.Errorf("DeleteKerberosRealm: %v", err)
	}
}

func TestDeleteKerberosRealm_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteKerberosRealm(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "deleting kerberos realm") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// Keytab
// =============================================================================

func TestGetKerberosKeytab(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.keytab.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "host", "file": "QUJDRA=="}, nil
	})
	c, _ := ts.NewClient(ctx)
	k, err := c.GetKerberosKeytab(ctx, 1)
	if err != nil {
		t.Fatalf("GetKerberosKeytab: %v", err)
	}
	if k.Name != "host" {
		t.Errorf("got %+v", k)
	}
}

func TestGetKerberosKeytab_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKerberosKeytab(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting kerberos keytab") {
		t.Errorf("got %v", err)
	}
}

func TestGetKerberosKeytab_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKerberosKeytab(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestCreateKerberosKeytab(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.keytab.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "name": "service"}, nil
	})
	c, _ := ts.NewClient(ctx)
	k, err := c.CreateKerberosKeytab(ctx, &types.KerberosKeytabCreateRequest{
		Name: "service", File: "QUJDRA==",
	})
	if err != nil {
		t.Fatalf("CreateKerberosKeytab: %v", err)
	}
	if k.ID != 9 {
		t.Errorf("got %+v", k)
	}
}

func TestCreateKerberosKeytab_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateKerberosKeytab(ctx, &types.KerberosKeytabCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating kerberos keytab") {
		t.Errorf("got %v", err)
	}
}

func TestCreateKerberosKeytab_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateKerberosKeytab(ctx, &types.KerberosKeytabCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateKerberosKeytab(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.keytab.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	n := "renamed"
	k, err := c.UpdateKerberosKeytab(ctx, 9, &types.KerberosKeytabUpdateRequest{Name: &n})
	if err != nil {
		t.Fatalf("UpdateKerberosKeytab: %v", err)
	}
	if k.Name != "renamed" {
		t.Errorf("got %+v", k)
	}
}

func TestUpdateKerberosKeytab_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKerberosKeytab(ctx, 9, &types.KerberosKeytabUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating kerberos keytab") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateKerberosKeytab_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKerberosKeytab(ctx, 9, &types.KerberosKeytabUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteKerberosKeytab(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "kerberos.keytab.delete" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteKerberosKeytab(ctx, 9); err != nil {
		t.Errorf("DeleteKerberosKeytab: %v", err)
	}
}

func TestDeleteKerberosKeytab_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteKerberosKeytab(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "deleting kerberos keytab") {
		t.Errorf("got %v", err)
	}
}
