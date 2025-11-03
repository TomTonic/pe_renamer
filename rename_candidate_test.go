package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	misc "pe_renamer/misc"
	"runtime"
	"strings"
	"testing"
)

func Test_RenameCandidate_DryRunNotVerbose(t *testing.T) {
	td := misc.CreateTestDir(t)
	defer func() { _ = os.RemoveAll(td) }()

	// copy fixture into td
	misc.CopyFromTestdata(t, "puttywin64x64", td, "")

	// capture directory tree before
	before, err := misc.DirTree(t, td, false)
	if err != nil {
		t.Fatalf("DirTree before failed: %v", err)
	}

	// candidate should reference the obfuscated name (no extension)
	cand := renamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "putty.exe",
	}

	expectedOutput := "Renaming " + filepath.Join(td, "puttywin64x64") + " → " + filepath.Join(td, "puttywin64x64", "putty.exe") + "\n"

	var buf bytes.Buffer
	renameCandidate(cand, false, true, false, false, &buf, io.Discard)

	out := buf.String()
	if out != expectedOutput {
		t.Fatalf("unexpected dry-run output.\nexpected: %s\ngot:     %s", expectedOutput, out)
	}

	// capture directory tree after and assert no changes as this is a dry-run
	after, err := misc.DirTree(t, td, false)
	if err != nil {
		t.Fatalf("DirTree after failed: %v", err)
	}
	if !before.Equals(after) {
		t.Fatalf("temp dir changed during dry-run (before=%v after=%v)", before.ToArray(), after.ToArray())
	}
}

func Test_RenameCandidate_DryRunVerbose(t *testing.T) {
	td := misc.CreateTestDir(t)
	defer func() { _ = os.RemoveAll(td) }()

	// copy fixture into td
	misc.CopyFromTestdata(t, "puttywin64x64", td, "")

	// capture directory tree before
	before, err := misc.DirTree(t, td, false)
	if err != nil {
		t.Fatalf("DirTree before failed: %v", err)
	}

	// candidate should reference the obfuscated name (no extension)
	cand := renamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "putty.exe",
		ExtMatches:   true,
		Similarity:   95.0,
	}

	expectedOutput := "Renaming " + filepath.Join(td, "puttywin64x64") + " → " + filepath.Join(td, "puttywin64x64", "putty.exe") + "\n"

	var buf bytes.Buffer
	renameCandidate(cand, false, true, false, false, &buf, io.Discard)

	out := buf.String()
	if out != expectedOutput {
		t.Fatalf("unexpected dry-run output.\nexpected: %s\ngot:     %s", expectedOutput, out)
	}

	// capture directory tree after and assert no changes as this is a dry-run
	after, err := misc.DirTree(t, td, false)
	if err != nil {
		t.Fatalf("DirTree after failed: %v", err)
	}
	if !before.Equals(after) {
		t.Fatalf("temp dir changed during dry-run (before=%v after=%v)", before.ToArray(), after.ToArray())
	}
}

func Test_RenameCandidate_Apply(t *testing.T) {
	td := misc.CreateTestDir(t)
	defer func() { _ = os.RemoveAll(td) }()

	// copy fixture into td
	misc.CopyFromTestdata(t, "puttywin64x64", td, "")

	sha256Original, err := misc.FileSHA256(t, filepath.Join(td, "puttywin64x64"))
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}

	// capture directory tree and make sure there is only one file system entry
	before, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("os.ReadDir before failed: %v", err)
	}

	if before == nil || len(before) != 1 {
		t.Fatalf("unexpected before listing: %v", before)
	}

	cand := renamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "putty.exe",
	}

	var buf bytes.Buffer
	renameCandidate(cand, false, false, false, false, &buf, io.Discard)

	// capture directory tree after and assert there is again only one file system entry
	after, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("os.ReadDir after failed: %v", err)
	}
	if after == nil || len(after) != 1 {
		t.Fatalf("unexpected after listing: %v", after)
	}

	// make sure the renamed file exists
	renamedPath := filepath.Join(td, "puttywin64x64", "putty.exe")
	fi, err := os.Stat(renamedPath)
	if err != nil {
		t.Fatalf("renamed file does not exist: %v", err)
	}

	// make sure it's a file
	if fi.IsDir() {
		t.Fatalf("renamed path is a directory, expected file: %s", renamedPath)
	}

	// make sure no other files exist in td/puttywin64x64/
	entries, err := os.ReadDir(filepath.Join(td, "puttywin64x64"))
	if err != nil {
		t.Fatalf("reading renamed dir entries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected number of entries in renamed dir: %v", entries)
	}

	// make sure the original file from the testdata direcory, puttywin64x64 and the renamed file putty.exe are equal
	sha256NewFile, err := misc.FileSHA256(t, filepath.Join(td, "puttywin64x64", "putty.exe"))
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}
	if sha256Original != sha256NewFile {
		t.Fatalf("renamed file data does not match original file data (original sha256=%s new sha256=%s)", sha256Original, sha256NewFile)
	}
}

func Test_RenameCandidate_Apply_JustExt(t *testing.T) {
	td := misc.CreateTestDir(t)
	defer func() { _ = os.RemoveAll(td) }()

	// copy fixture into td
	misc.CopyFromTestdata(t, "puttywin64x64", td, "")

	sha256Original, err := misc.FileSHA256(t, filepath.Join(td, "puttywin64x64"))
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}

	// capture directory tree and make sure there is only one file system entry
	before, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("os.ReadDir before failed: %v", err)
	}

	if before == nil || len(before) != 1 {
		t.Fatalf("unexpected before listing: %v", before)
	}

	cand := renamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "somethingrandom",
	}

	var buf bytes.Buffer
	renameCandidate(cand, false, false, true, false, &buf, io.Discard)

	// capture directory tree after and assert there is again only one file system entry
	after, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("os.ReadDir after failed: %v", err)
	}
	if after == nil || len(after) != 1 {
		t.Fatalf("unexpected after listing: %v", after)
	}

	// make sure the renamed file exists at top-level
	renamedPath := filepath.Join(td, "somethingrandom")
	fi, err := os.Stat(renamedPath)
	if err != nil {
		t.Fatalf("renamed file does not exist: %v", err)
	}

	// make sure it's a file
	if fi.IsDir() {
		t.Fatalf("renamed path is a directory, expected file: %s", renamedPath)
	}

	// make sure original directory no longer exists
	if _, err := os.Stat(filepath.Join(td, "puttywin64x64")); err == nil {
		t.Fatalf("original obfuscated path still exists, expected it to be moved: %s", filepath.Join(td, "puttywin64x64"))
	}

	// make sure the renamed file data matches original
	sha256NewFile, err := misc.FileSHA256(t, renamedPath)
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}
	if sha256Original != sha256NewFile {
		t.Fatalf("renamed file data does not match original file data (original sha256=%s new sha256=%s)", sha256Original, sha256NewFile)
	}
}

func Test_RenameCandidate_ReadOnly_ExtOnly(t *testing.T) {
	// filesystem permission semantics differ on Windows; skip this test there
	if runtime.GOOS == "windows" {
		t.Skip("skipping read-only permission test on Windows")
	}
	td := misc.CreateTestDir(t)
	defer func() { _ = os.RemoveAll(td) }()

	// copy fixture into td
	misc.CopyFromTestdata(t, "puttywin64x64", td, "")

	// make directory read-only
	if err := os.Chmod(td, 0o555); err != nil {
		t.Fatalf("chmod readonly failed: %v", err)
	}
	// restore permissions for cleanup
	defer func() { _ = os.Chmod(td, 0o755) }()

	cand := renamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "somethingrandom",
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	renameCandidate(cand, false, false, true, false, &outBuf, &errBuf)

	if !strings.Contains(errBuf.String(), "permission denied") {
		t.Fatalf("expected permission denied in stderr, got: %s", errBuf.String())
	}

	// original file should still exist
	if _, err := os.Stat(filepath.Join(td, "puttywin64x64")); err != nil {
		t.Fatalf("expected original file to still exist, stat error: %v", err)
	}
}

func Test_RenameCandidate_ReadOnly_FullRename(t *testing.T) {
	// filesystem permission semantics differ on Windows; skip this test there
	if runtime.GOOS == "windows" {
		t.Skip("skipping read-only permission test on Windows")
	}
	td := misc.CreateTestDir(t)
	defer func() { _ = os.RemoveAll(td) }()

	// copy fixture into td
	misc.CopyFromTestdata(t, "puttywin64x64", td, "")

	// make directory read-only
	if err := os.Chmod(td, 0o555); err != nil {
		t.Fatalf("chmod readonly failed: %v", err)
	}
	// restore permissions for cleanup
	defer func() { _ = os.Chmod(td, 0o755) }()

	cand := renamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "putty.exe",
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	renameCandidate(cand, false, false, false, false, &outBuf, &errBuf)

	if !strings.Contains(errBuf.String(), "permission denied") {
		t.Fatalf("expected permission denied in stderr, got: %s", errBuf.String())
	}

	// original file should still exist
	if _, err := os.Stat(filepath.Join(td, "puttywin64x64")); err != nil {
		t.Fatalf("expected original file to still exist, stat error: %v", err)
	}
}
