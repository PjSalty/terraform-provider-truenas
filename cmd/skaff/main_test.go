package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withSilentIO redirects stdout, stderr, and exitFn so tests don't dump to
// the terminal. The returned buffers capture whatever the program wrote;
// lastExit captures the last exit code that exitFn was called with.
func withSilentIO(t *testing.T) (stdoutBuf, stderrBuf *bytes.Buffer, lastExit *int) {
	t.Helper()
	stdoutBuf = &bytes.Buffer{}
	stderrBuf = &bytes.Buffer{}
	origStdout := stdout
	origStderr := stderr
	origExit := exitFn
	stdout = stdoutBuf
	stderr = stderrBuf
	lastExit = new(int)
	exitFn = func(code int) { *lastExit = code }
	t.Cleanup(func() {
		stdout = origStdout
		stderr = origStderr
		exitFn = origExit
	})
	return
}

// inTempDir runs fn from a fresh temp dir, restoring cwd afterward.
func inTempDir(t *testing.T, fn func()) {
	t.Helper()
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	fn()
}

func TestSnakeToCamel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"ftp_config", "FTPConfig"},
		{"smb_config", "SMBConfig"},
		{"iscsi_target", "IscsiTarget"},
		{"vm_device", "VMDevice"},
		{"api_key", "APIKey"},
		{"http_proxy", "HTTPProxy"},
		{"ups", "UPS"},
		{"single", "Single"},
		{"multi_word_name", "MultiWordName"},
		{"", ""},
		{"nvme_host", "NVMeHost"},
	}
	for _, c := range cases {
		got := snakeToCamel(c.in)
		if got != c.want {
			t.Errorf("snakeToCamel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildData(t *testing.T) {
	d := buildData("ftp_config")
	if d.Name != "ftp_config" {
		t.Errorf("Name: got %q", d.Name)
	}
	if d.CamelName != "FTPConfig" {
		t.Errorf("CamelName: got %q", d.CamelName)
	}
	if d.TypeName != "truenas_ftp_config" {
		t.Errorf("TypeName: got %q", d.TypeName)
	}
	if d.Description == "" {
		t.Error("Description must be non-empty")
	}
}

func TestScaffoldResource_Success(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		if err := scaffoldResource("test_resource"); err != nil {
			t.Fatalf("scaffoldResource: %v", err)
		}
		for _, want := range []string{
			filepath.Join("internal", "resources", "test_resource.go"),
			filepath.Join("internal", "client", "test_resource.go"),
			filepath.Join("docs", "resources", "test_resource.md"),
			filepath.Join("examples", "resources", "truenas_test_resource", "resource.tf"),
			filepath.Join("examples", "resources", "truenas_test_resource", "import.sh"),
		} {
			data, err := os.ReadFile(want)
			if err != nil {
				t.Errorf("expected file %s: %v", want, err)
				continue
			}
			if len(data) == 0 {
				t.Errorf("file %s is empty", want)
			}
			if strings.Contains(string(data), "{{") {
				t.Errorf("file %s still contains unrendered template tags", want)
			}
		}
	})
}

func TestScaffoldResource_RefusesOverwrite(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		if err := scaffoldResource("test_resource"); err != nil {
			t.Fatalf("first scaffold: %v", err)
		}
		if err := scaffoldResource("test_resource"); err == nil {
			t.Fatal("expected error on second scaffold (overwrite), got nil")
		}
	})
}

func TestScaffoldResource_StatError(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		// Create a path where a stat can neither succeed as "exists" nor
		// return os.ErrNotExist: point at a broken symlink, whose Stat
		// returns ENOENT (mapped to ErrNotExist). To get a non-ErrNotExist
		// stat error we point through a non-directory parent, which
		// returns ENOTDIR.
		if err := os.WriteFile("not_a_dir", []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		// Build a custom writeAll with a target that stats through the file.
		target := scaffoldTarget{
			path: filepath.Join("not_a_dir", "child"),
			tmpl: "hello",
		}
		err := writeAll([]scaffoldTarget{target}, buildData("foo"))
		if err == nil {
			t.Fatal("expected ENOTDIR-wrapped stat error, got nil")
		}
		if !strings.Contains(err.Error(), "stat") {
			t.Errorf("expected stat error, got %v", err)
		}
	})
}

func TestScaffoldDataSource_Success(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		if err := scaffoldDataSource("test_ds"); err != nil {
			t.Fatalf("scaffoldDataSource: %v", err)
		}
		for _, want := range []string{
			filepath.Join("internal", "datasources", "test_ds.go"),
			filepath.Join("docs", "data-sources", "test_ds.md"),
			filepath.Join("examples", "data-sources", "truenas_test_ds", "data-source.tf"),
		} {
			if _, err := os.Stat(want); err != nil {
				t.Errorf("expected file %s: %v", want, err)
			}
		}
	})
}

func TestScaffoldDataSource_RefusesOverwrite(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		if err := scaffoldDataSource("test_ds"); err != nil {
			t.Fatalf("first scaffold: %v", err)
		}
		if err := scaffoldDataSource("test_ds"); err == nil {
			t.Fatal("expected error on second scaffold, got nil")
		}
	})
}

func TestScaffoldResource_MkdirFails(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		// Block creation of the 'internal' directory by putting a file there.
		if err := os.WriteFile("internal", []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		err := scaffoldResource("blocked")
		if err == nil {
			t.Fatal("expected mkdir failure, got nil")
		}
	})
}

func TestWriteAll_BadTemplate(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		// Template with a syntax error triggers template.Parse failure.
		err := writeAll([]scaffoldTarget{{
			path: "out.txt",
			tmpl: `{{ .Unterminated `,
		}}, buildData("x"))
		if err == nil || !strings.Contains(err.Error(), "parse template") {
			t.Fatalf("expected parse template error, got %v", err)
		}
	})
}

func TestWriteAll_MkdirFails(t *testing.T) {
	_, _, _ = withSilentIO(t)
	orig := osMkdirAll
	osMkdirAll = func(_ string, _ os.FileMode) error {
		return errors.New("injected mkdir failure")
	}
	t.Cleanup(func() { osMkdirAll = orig })
	inTempDir(t, func() {
		err := writeAll([]scaffoldTarget{{
			path: filepath.Join("sub", "child"),
			tmpl: "hello",
		}}, buildData("x"))
		if err == nil || !strings.Contains(err.Error(), "mkdir") {
			t.Fatalf("expected mkdir error, got %v", err)
		}
	})
}

func TestWriteAll_CreateFails(t *testing.T) {
	_, _, _ = withSilentIO(t)
	orig := osCreate
	osCreate = func(_ string) (*os.File, error) {
		return nil, errors.New("injected create failure")
	}
	t.Cleanup(func() { osCreate = orig })
	inTempDir(t, func() {
		err := writeAll([]scaffoldTarget{{
			path: "child",
			tmpl: "hello",
		}}, buildData("x"))
		if err == nil || !strings.Contains(err.Error(), "create") {
			t.Fatalf("expected create error, got %v", err)
		}
	})
}

func TestWriteAll_ExecuteFails(t *testing.T) {
	_, _, _ = withSilentIO(t)
	inTempDir(t, func() {
		// Template that references an undefined field — Parse succeeds
		// (syntax is valid), Execute fails with "can't evaluate field".
		err := writeAll([]scaffoldTarget{{
			path: "out.txt",
			tmpl: "{{.DoesNotExist}}",
		}}, buildData("x"))
		if err == nil || !strings.Contains(err.Error(), "render") {
			t.Fatalf("expected render error, got %v", err)
		}
	})
}

// --- run / main / fatal tests ------------------------------------------

func TestRun_ResourceSuccess(t *testing.T) {
	_, _, _ = withSilentIO(t)
	var code int
	inTempDir(t, func() {
		code = run([]string{"resource", "test_r"})
	})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_DataSourceSuccess(t *testing.T) {
	_, _, _ = withSilentIO(t)
	var code int
	inTempDir(t, func() {
		code = run([]string{"datasource", "test_ds"})
	})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_DataSourceAlias(t *testing.T) {
	_, _, _ = withSilentIO(t)
	var code int
	inTempDir(t, func() {
		code = run([]string{"data-source", "test_ds_alias"})
	})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_BadName(t *testing.T) {
	_, stderrBuf, _ := withSilentIO(t)
	code := run([]string{"resource", "Bad-Name"})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderrBuf.String(), "snake_case") {
		t.Errorf("expected snake_case error, got %q", stderrBuf.String())
	}
}

func TestRun_BadKind(t *testing.T) {
	_, _, _ = withSilentIO(t)
	code := run([]string{"unknown_kind", "foo"})
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestRun_NotEnoughArgs(t *testing.T) {
	_, _, _ = withSilentIO(t)
	code := run([]string{"resource"})
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestRun_FlagParseError(t *testing.T) {
	_, _, _ = withSilentIO(t)
	// Unknown flag triggers fs.Parse error → returns 2.
	code := run([]string{"-unknown"})
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestRun_ResourceMkdirFail(t *testing.T) {
	_, _, _ = withSilentIO(t)
	var code int
	inTempDir(t, func() {
		if err := os.WriteFile("internal", []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		code = run([]string{"resource", "blocked"})
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestRun_DataSourceMkdirFail(t *testing.T) {
	_, _, _ = withSilentIO(t)
	var code int
	inTempDir(t, func() {
		if err := os.WriteFile("internal", []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		code = run([]string{"datasource", "blocked"})
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestMain_CallsRun(t *testing.T) {
	_, _, lastExit := withSilentIO(t)
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	inTempDir(t, func() {
		os.Args = []string{"skaff", "resource", "test_main"}
		main()
	})
	if *lastExit != 0 {
		t.Errorf("expected exit 0 from main(), got %d", *lastExit)
	}
}

func TestFatal(t *testing.T) {
	_, stderrBuf, _ := withSilentIO(t)
	fatal(errors.New("something broke"))
	if !strings.Contains(stderrBuf.String(), "something broke") {
		t.Errorf("expected error in stderr, got %q", stderrBuf.String())
	}
}
