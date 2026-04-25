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
	keys := []string{"TF_ACC", "TRUENAS_URL", "TRUENAS_API_KEY", "TRUENAS_INSECURE_SKIP_VERIFY"}
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
