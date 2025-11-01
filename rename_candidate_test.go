package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"pe_renamer/testhelpers"
)

func Test_RenameCandidate_DryRunNotVerbose(t *testing.T) {
	td := testhelpers.CreateTestDir(t)
	defer os.RemoveAll(td)

	// copy fixture into td
	testhelpers.CopyFromTestdata(t, "puttywin64x64", td, "")

	// capture directory tree before
	before, err := testhelpers.DirTree(t, td)
	if err != nil {
		t.Fatalf("DirTree before failed: %v", err)
	}

	// candidate should reference the obfuscated name (no extension)
	cand := RenamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "putty.exe",
	}

	expectedOutput := "Renaming " + filepath.Join(td, "puttywin64x64") + " → " + filepath.Join(td, "puttywin64x64", "putty.exe") + "\n"

	var buf bytes.Buffer
	if err := renameCandidate(&buf, cand, false, true); err != nil {
		t.Fatalf("renameCandidate dry-run returned error: %v", err)
	}

	out := buf.String()
	if out != expectedOutput {
		t.Fatalf("unexpected dry-run output.\nexpected: %s\ngot:     %s", expectedOutput, out)
	}

	// capture directory tree after and assert no changes as this is a dry-run
	after, err := testhelpers.DirTree(t, td)
	if err != nil {
		t.Fatalf("DirTree after failed: %v", err)
	}
	if !before.Equals(after) {
		t.Fatalf("temp dir changed during dry-run (before=%v after=%v)", before.ToArray(), after.ToArray())
	}
}

func Test_RenameCandidate_DryRunVerbose(t *testing.T) {
	td := testhelpers.CreateTestDir(t)
	defer os.RemoveAll(td)

	// copy fixture into td
	testhelpers.CopyFromTestdata(t, "puttywin64x64", td, "")

	// capture directory tree before
	before, err := testhelpers.DirTree(t, td)
	if err != nil {
		t.Fatalf("DirTree before failed: %v", err)
	}

	// candidate should reference the obfuscated name (no extension)
	cand := RenamingCandidate{
		Path:                        td,
		OriginalName:                "puttywin64x64",
		NewName:                     "putty.exe",
		matching_extension:          true,
		editing_distance_percentage: 95.0,
	}

	expectedOutput := "File: puttywin64x64\n      new name: putty.exe\n      similarity: 95.0%\n      extension matches: true\nRenaming " + filepath.Join(td, "puttywin64x64") + " → " + filepath.Join(td, "puttywin64x64", "putty.exe") + "\n"

	var buf bytes.Buffer
	if err := renameCandidate(&buf, cand, true, true); err != nil {
		t.Fatalf("renameCandidate dry-run returned error: %v", err)
	}

	out := buf.String()
	if out != expectedOutput {
		t.Fatalf("unexpected dry-run output.\nexpected: %s\ngot:     %s", expectedOutput, out)
	}

	// capture directory tree after and assert no changes as this is a dry-run
	after, err := testhelpers.DirTree(t, td)
	if err != nil {
		t.Fatalf("DirTree after failed: %v", err)
	}
	if !before.Equals(after) {
		t.Fatalf("temp dir changed during dry-run (before=%v after=%v)", before.ToArray(), after.ToArray())
	}
}

func Test_RenameCandidate_Apply(t *testing.T) {
	td := testhelpers.CreateTestDir(t)
	defer os.RemoveAll(td)

	// copy fixture into td
	testhelpers.CopyFromTestdata(t, "puttywin64x64", td, "")

	sha256Original, err := testhelpers.FileSHA256(filepath.Join(td, "puttywin64x64"))
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

	cand := RenamingCandidate{
		Path:         td,
		OriginalName: "puttywin64x64",
		NewName:      "putty.exe",
	}

	var buf bytes.Buffer
	if err := renameCandidate(&buf, cand, false, false); err != nil {
		t.Fatalf("renameCandidate apply returned error: %v", err)
	}

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
	sha256NewFile, err := testhelpers.FileSHA256(filepath.Join(td, "puttywin64x64", "putty.exe"))
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}
	if sha256Original != sha256NewFile {
		t.Fatalf("renamed file data does not match original file data (original sha256=%s new sha256=%s)", sha256Original, sha256NewFile)
	}
}
