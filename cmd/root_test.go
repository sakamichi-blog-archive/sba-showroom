package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	defer func() { _ = r.Close() }()

	f()

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(out)
}

func TestRun_VersionFlags(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		t.Run(arg, func(t *testing.T) {
			var code int
			out := captureStdout(t, func() { code = run([]string{arg}) })
			if code != 0 {
				t.Errorf("run(%q) exit code = %d, want 0", arg, code)
			}
			if !strings.Contains(out, "sba-showroom") {
				t.Errorf("run(%q) output = %q, want it to contain %q", arg, out, "sba-showroom")
			}
		})
	}
}

func TestRun_HelpFlags(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			var code int
			out := captureStdout(t, func() { code = run([]string{arg}) })
			if code != 0 {
				t.Errorf("run(%q) exit code = %d, want 0", arg, code)
			}
			if !strings.Contains(out, "Usage: sba-showroom") {
				t.Errorf("run(%q) output = %q, want it to contain usage", arg, out)
			}
		})
	}
}

func TestRun_NoArgs(t *testing.T) {
	var code int
	out := captureStdout(t, func() { code = run(nil) })
	if code != 1 {
		t.Errorf("run(nil) exit code = %d, want 1", code)
	}
	if !strings.Contains(out, "Usage: sba-showroom") {
		t.Errorf("run(nil) output = %q, want it to contain usage", out)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var code int
	out := captureStdout(t, func() { code = run([]string{"bogus"}) })
	if code != 1 {
		t.Errorf(`run(["bogus"]) exit code = %d, want 1`, code)
	}
	if !strings.Contains(out, "Usage: sba-showroom") {
		t.Errorf(`run(["bogus"]) output = %q, want it to contain usage`, out)
	}
}

func TestResolveVersion_Default(t *testing.T) {
	if got := resolveVersion(); got == "" {
		t.Error("resolveVersion() = \"\", want a non-empty version string")
	}
}
