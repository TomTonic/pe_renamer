package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var e2eBin string

// TestMain builds the binary once for all CLI e2e tests and removes it afterwards.
func TestMain(m *testing.M) {
	tmpdir, err := os.MkdirTemp("", "pe_renamer_e2e_")
	if err != nil {
		_, _ = os.Stderr.WriteString("failed to create tmpdir for e2e build: " + err.Error())
		os.Exit(2)
	}
	e2eBin = filepath.Join(tmpdir, "pe_renamer_e2e")
	cmd := exec.Command("go", "build", "-o", e2eBin, ".")
	cmd.Env = os.Environ()
	if b, err := cmd.CombinedOutput(); err != nil {
		_, _ = os.Stderr.WriteString("go build failed: " + err.Error() + "\noutput:\n" + string(b))
		_ = os.RemoveAll(tmpdir)
		os.Exit(2)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpdir)
	os.Exit(code)
}

func Test_CLI_VersionFlag(t *testing.T) {
	cmd := exec.Command(e2eBin, "--version")
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running version flag failed: %v\noutput:\n%s", err, string(b))
	}
	out := string(b)
	if !strings.Contains(out, "OS:") || !strings.Contains(out, "ARCH:") || !strings.Contains(out, "TAG:") {
		t.Fatalf("unexpected version output:\n%s", out)
	}
}

func Test_CLI_MissingPath(t *testing.T) {
	cmd := exec.Command(e2eBin)
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error when running without args, got none; output:\n%s", string(b))
	}
	out := string(b)
	if !strings.Contains(out, "path is required") {
		t.Fatalf("expected 'path is required' message, got:\n%s", out)
	}
}

func Test_CLI_DryRunOnNonPE(t *testing.T) {
	// use repository testdata/somepng which is not a PE file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	target := filepath.Join(wd, "testdata", "somepng")

	cmd := exec.Command(e2eBin, "-n", "-v", target)
	b, err := cmd.CombinedOutput()
	if err != nil {
		_ = err
	}
	out := string(b)
	if !strings.Contains(out, "File is not in PE format") {
		t.Fatalf("expected non-PE message in output, got:\n%s", out)
	}
}
