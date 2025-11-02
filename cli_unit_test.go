package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCli_VersionFlag(t *testing.T) {
	var out, errb strings.Builder
	code := runCli([]string{"--version"}, &out, &errb)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr=%s", code, errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "OS:") || !strings.Contains(s, "ARCH:") || !strings.Contains(s, "TAG:") {
		t.Fatalf("unexpected version output: %s", s)
	}
}

func TestRunCli_MissingPath(t *testing.T) {
	var out, errb strings.Builder
	code := runCli([]string{}, &out, &errb)
	if code == 0 {
		t.Fatalf("expected non-zero exit code when path missing; got 0")
	}
	if !strings.Contains(errb.String(), "path is required") {
		t.Fatalf("expected 'path is required' in stderr, got: %s", errb.String())
	}
}

func TestRunCli_DryRunNonPE(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	target := filepath.Join(wd, "testdata", "somepng")

	var out, errb strings.Builder
	code := runCli([]string{"-n", "-v", target}, &out, &errb)
	if code != 0 {
		t.Fatalf("expected exit 0 for dry-run non-PE, got %d; stderr=%s", code, errb.String())
	}
	combined := out.String() + "\n" + errb.String()
	if !strings.Contains(combined, "File is not in PE format") {
		t.Fatalf("expected non-PE message in output, got:\n%s", combined)
	}
}

func TestRunCli_NonexistentPath(t *testing.T) {
	var out, errb strings.Builder
	// create a temp dir and point to a child that does not exist
	td := t.TempDir()
	target := filepath.Join(td, "this-path-should-not-exist-12345")

	code := runCli([]string{target}, &out, &errb)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for nonexistent path; got 0")
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Fatalf("expected 'no such file or directory' in stderr, got: %s", errb.String())
	}
}
