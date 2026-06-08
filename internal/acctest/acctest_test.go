package acctest_test

import (
	"os"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
)

// envSandbox snapshots and clears the env variables touched by the
// helpers so each subtest starts from a known baseline. The restore
// callback reinstates the prior values even on failure.
func envSandbox(t *testing.T) func() {
	t.Helper()
	keys := []string{"TF_ACC", "TRUENAS_URL", "TRUENAS_API_KEY", "TRUENAS_INSECURE_SKIP_VERIFY", "TRUENAS_PROD_DENY"}
	prev := map[string]string{}
	for _, k := range keys {
		prev[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	return func() {
		for _, k := range keys {
			if v, ok := prev[k]; ok && v != "" {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}
}

func TestSkipMsg(t *testing.T) {
	if !strings.Contains(acctest.SkipMsg, "TF_ACC") {
		t.Fatalf("SkipMsg should mention TF_ACC, got %q", acctest.SkipMsg)
	}
}

func TestSkipIfNoAcc_Skips(t *testing.T) {
	restore := envSandbox(t)
	defer restore()

	// t.Skip calls runtime.Goexit so we have to run SkipIfNoAcc in a
	// throwaway goroutine and observe the skip flag afterwards.
	inner := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer func() {
			_ = recover()
			close(done)
		}()
		acctest.SkipIfNoAcc(inner)
	}()
	<-done
	if !inner.Skipped() {
		t.Fatalf("TF_ACC unset: inner test should be marked Skipped")
	}
}

func TestSkipIfNoAcc_DoesNotSkip(t *testing.T) {
	restore := envSandbox(t)
	defer restore()
	t.Setenv("TF_ACC", "1")

	inner := &testing.T{}
	skipped := acctest.SkipIfNoAcc(inner)
	if skipped {
		t.Fatalf("TF_ACC=1: SkipIfNoAcc must return false")
	}
	if inner.Skipped() {
		t.Fatalf("TF_ACC=1: inner test must not be skipped")
	}
}

func TestPreCheck_OK(t *testing.T) {
	restore := envSandbox(t)
	defer restore()
	t.Setenv("TRUENAS_URL", "https://example.invalid")
	t.Setenv("TRUENAS_API_KEY", "key")

	inner := &testing.T{}
	acctest.PreCheck(inner)
	if inner.Failed() {
		t.Fatalf("PreCheck should not fail when env is configured")
	}
}

func TestPreCheck_MissingURL(t *testing.T) {
	restore := envSandbox(t)
	defer restore()
	t.Setenv("TRUENAS_API_KEY", "key")

	inner := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer func() {
			_ = recover() // t.Fatal inside goroutine triggers runtime.Goexit
			close(done)
		}()
		acctest.PreCheck(inner)
	}()
	<-done
	if !inner.Failed() {
		t.Fatalf("missing URL: PreCheck must call t.Fatal")
	}
}

func TestPreCheck_MissingKey(t *testing.T) {
	restore := envSandbox(t)
	defer restore()
	t.Setenv("TRUENAS_URL", "https://example.invalid")

	inner := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer func() {
			_ = recover()
			close(done)
		}()
		acctest.PreCheck(inner)
	}()
	<-done
	if !inner.Failed() {
		t.Fatalf("missing API key: PreCheck must call t.Fatal")
	}
}

func TestClient_MissingEnv(t *testing.T) {
	restore := envSandbox(t)
	defer restore()

	if _, err := acctest.Client(); err == nil {
		t.Fatalf("Client must error when env vars are unset")
	}
}

func TestClient_OK(t *testing.T) {
	restore := envSandbox(t)
	defer restore()
	t.Setenv("TRUENAS_URL", "https://example.invalid")
	t.Setenv("TRUENAS_API_KEY", "key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	if c == nil {
		t.Fatalf("Client returned nil")
	}
}

func TestCtx(t *testing.T) {
	ctx, cancel := acctest.Ctx()
	defer cancel()
	if ctx == nil {
		t.Fatalf("Ctx returned nil context")
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatalf("Ctx must return a context with a deadline")
	}
}

func TestRandomName(t *testing.T) {
	a := acctest.RandomName("tf-acc")
	if !strings.HasPrefix(a, "tf-acc-") {
		t.Fatalf("RandomName should start with prefix-: %q", a)
	}
	b := acctest.RandomName("tf-acc")
	if a == b {
		t.Fatalf("RandomName should produce distinct values: %q == %q", a, b)
	}
}

func TestShortSuffix(t *testing.T) {
	s := acctest.ShortSuffix()
	if s < 0 || s >= 1000000 {
		t.Fatalf("ShortSuffix out of range: %d", s)
	}
}

func TestPIDSuffix(t *testing.T) {
	s := acctest.PIDSuffix()
	if s < 0 || s >= 1000000 {
		t.Fatalf("PIDSuffix out of range: %d", s)
	}
}

// TestClient_RejectsProdHost verifies the prod-deny safety rail
// refuses to build a client targeting any host listed in
// TRUENAS_PROD_DENY. Walks the most likely operator-error paths:
//
//   - Default deny list catches the homelab production hostname.
//   - Explicit denylist override is honored.
//   - Hostname matching is case-insensitive.
//   - The override-to-empty path lets a determined operator disable
//     the check (after a clear acctest documentation gate).
//   - Wrong-shape URLs surface a clean error instead of silently
//     bypassing the check.
//
// The destructive default — building a client at all — is gated
// behind these checks. Each subtest snapshots/restores env so they
// can run in any order without polluting siblings.
func TestClient_RejectsProdHost(t *testing.T) {
	t.Run("default-deny catches default prod host", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		t.Setenv("TRUENAS_URL", "https://"+acctest.DefaultProdDeny)
		t.Setenv("TRUENAS_API_KEY", "k")
		if _, err := acctest.Client(); err == nil ||
			!strings.Contains(err.Error(), "TRUENAS_PROD_DENY") {
			t.Errorf("expected prod-deny rejection, got: %v", err)
		}
	})

	t.Run("explicit denylist override is honored", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		t.Setenv("TRUENAS_URL", "https://my-lab-prod.example")
		t.Setenv("TRUENAS_API_KEY", "k")
		t.Setenv("TRUENAS_PROD_DENY", "my-lab-prod.example, other-prod.example")
		if _, err := acctest.Client(); err == nil ||
			!strings.Contains(err.Error(), "my-lab-prod.example") {
			t.Errorf("expected rejection naming my-lab-prod.example, got: %v", err)
		}
	})

	t.Run("hostname match is case-insensitive", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		t.Setenv("TRUENAS_URL", "https://MY-LAB-PROD.EXAMPLE")
		t.Setenv("TRUENAS_API_KEY", "k")
		t.Setenv("TRUENAS_PROD_DENY", "my-lab-prod.example")
		if _, err := acctest.Client(); err == nil ||
			!strings.Contains(err.Error(), "TRUENAS_PROD_DENY") {
			t.Errorf("expected case-insensitive rejection, got: %v", err)
		}
	})

	t.Run("empty deny-list disables the check", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		t.Setenv("TRUENAS_URL", "https://"+acctest.DefaultProdDeny)
		t.Setenv("TRUENAS_API_KEY", "k")
		t.Setenv("TRUENAS_PROD_DENY", "")
		// Client() still has to construct a real *client.Client; we
		// don't assert success there because the URL may not resolve
		// — only that the prod-deny error is no longer in the path.
		_, err := acctest.Client()
		if err != nil && strings.Contains(err.Error(), "TRUENAS_PROD_DENY") {
			t.Errorf("empty TRUENAS_PROD_DENY should disable the check; got: %v", err)
		}
	})

	t.Run("non-prod URL passes through", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		t.Setenv("TRUENAS_URL", "https://test-truenas.lab.example")
		t.Setenv("TRUENAS_API_KEY", "k")
		// We expect no prod-deny error; the underlying client.NewWith
		// Options may still produce an error if it does anything
		// network-bound, but that's not what this test is asserting.
		_, err := acctest.Client()
		if err != nil && strings.Contains(err.Error(), "TRUENAS_PROD_DENY") {
			t.Errorf("non-prod URL should pass through; got: %v", err)
		}
	})

	t.Run("malformed URL surfaces a clean error", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		t.Setenv("TRUENAS_URL", "://no-scheme")
		t.Setenv("TRUENAS_API_KEY", "k")
		_, err := acctest.Client()
		if err == nil {
			t.Fatal("expected error on malformed URL")
		}
		// Either parse error OR "no hostname" — both are acceptable
		// outcomes that prevent the destructive default.
		if !strings.Contains(err.Error(), "not a valid URL") &&
			!strings.Contains(err.Error(), "no hostname") {
			t.Errorf("expected URL parse error, got: %v", err)
		}
	})

	t.Run("URL parses but has no hostname", func(t *testing.T) {
		restore := envSandbox(t)
		defer restore()
		// "https:/path" — url.Parse succeeds (single slash makes it
		// a path-only URL with empty host) but Hostname() returns "",
		// triggering the no-hostname guard inside assertNotProd.
		t.Setenv("TRUENAS_URL", "https:/path-only")
		t.Setenv("TRUENAS_API_KEY", "k")
		_, err := acctest.Client()
		if err == nil {
			t.Fatal("expected error on URL with no hostname")
		}
		if !strings.Contains(err.Error(), "no hostname") {
			t.Errorf("expected 'no hostname' error, got: %v", err)
		}
	})
}
