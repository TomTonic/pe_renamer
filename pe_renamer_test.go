package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pe_renamer/testhelpers"

	set3 "github.com/TomTonic/Set3"
)

// RunCasesAndCheck runs Run() in-process against the provided FixtureCase slice.
// It returns stdout, stderr and the testdir (caller must cleanup).
func RunCasesAndCheck(t *testing.T, cases []testhelpers.FixtureObject) {
	t.Helper()
	td := t.TempDir()
	defer os.RemoveAll(td)

	// prepare fixtures
	testhelpers.CopyCasesToDir(t, cases, td)

	// capture directory tree before
	beforeDirTree, err := testhelpers.DirTree(t, td)
	if err != nil {
		t.Fatalf("DirTree before failed: %v", err)
	}
	if int(beforeDirTree.Size()) != len(cases) {
		t.Fatalf("unexpected number of files in test dir before Run; expected %d, got %v", len(cases), beforeDirTree)
	}

	var stdout, stderr strings.Builder
	if err := Run(td, true, false, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v\nstderr: %s", err, stderr.String())
	}

	outStr := stdout.String()
	errStr := stderr.String()

	// capture directory tree before
	afterDirTree, err := testhelpers.DirTree(t, td)
	if err != nil {
		t.Fatalf("DirTree before failed: %v", err)
	}

	// check optional regexes per case
	for _, c := range cases {
		if c.StdoutRegex != nil && !c.StdoutRegex.MatchString(outStr) {
			t.Fatalf("stdout did not match regex for fixture %s; regex=%v\nstdout:\n%s", c.BinFile, c.StdoutRegex, outStr)
		}
		if c.StderrRegex != nil && !c.StderrRegex.MatchString(errStr) {
			t.Fatalf("stderr did not match regex for fixture %s; regex=%v\nstderr:\n%s", c.BinFile, c.StderrRegex, errStr)
		}
	}

	// assert expected files exist

	// first build expected dir tree
	expectedDirTree := set3.Empty[string]()
	for _, c := range cases {
		//expectedPath := strings.TrimPrefix(c.ExpectedFileName, "./")
		expectedPath := filepath.Clean(filepath.FromSlash(c.ExpectedFileName))
		parts := strings.Split(expectedPath, string(os.PathSeparator))
		current := ""
		for i, part := range parts {
			current = filepath.Join(current, part)
			currentNormalized := filepath.ToSlash(current)
			if i < len(parts)-1 {
				expectedDirTree.Add("D " + currentNormalized)
			} else {
				expectedDirTree.Add("F " + currentNormalized)
			}
		}
	}

	//expectedDirTree := set3.FromArray([]string{"D sqlite3win32x86", "F sqlite3win32x86\\sqlite3.dll"})

	if !expectedDirTree.Equals(afterDirTree) {
		t.Fatalf("unexpected directory tree after Run; expected=%v got=%v", expectedDirTree.ToArray(), afterDirTree.ToArray())
	}
}

func Test_Sqlite3DLL_Rename(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "./sqlite3win32x86",
			ExpectedFileName:   "./sqlite3win32x86/sqlite3.dll",
		},
		{
			BinFile:            "sqlite3win64x64",
			ObfuscatedFileName: "./sqlite3win64x64",
			ExpectedFileName:   "./sqlite3win64x64/sqlite3.dll",
		},
		{
			BinFile:            "sqlite3win64arm",
			ObfuscatedFileName: "./sqlite3win64arm",
			ExpectedFileName:   "./sqlite3win64arm/sqlite3.dll",
		},
	}

	RunCasesAndCheck(t, cases)
}

func Test_Log4netDLL_Rename(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "log4netdotnet20",
			ObfuscatedFileName: "./log4netdotnet20",
			ExpectedFileName:   "./log4netdotnet20/log4net.dll",
		},
		{
			BinFile:            "log4netdotnet462",
			ObfuscatedFileName: "./log4netdotnet462",
			ExpectedFileName:   "./log4netdotnet462/log4net.dll",
		},
	}

	RunCasesAndCheck(t, cases)
}
