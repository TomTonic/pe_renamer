package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"pe_renamer/testhelpers"

	set3 "github.com/TomTonic/Set3"
)

// RunCasesAndCheck runs Run() in-process against the provided FixtureCase slice.
// It returns stdout, stderr and the testdir (caller must cleanup).
func RunCasesAndCheck(t *testing.T, cases []testhelpers.FixtureObject, verbose bool, dryRun bool, justExt bool) {
	//t.Helper()
	td := t.TempDir()
	defer os.RemoveAll(td)

	// prepare fixtures
	testhelpers.CopyCasesToDir(t, cases, td)

	var stdout, stderr strings.Builder
	if err := Run(&stdout, &stderr, td, verbose, dryRun, justExt); err != nil {
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
			t.Fatalf("stdout did not match regex for fixture %s.\nregex:  %v\nstdout: %s\n", c.BinFile, c.StdoutRegex, outStr)
		}
		if c.StderrRegex != nil && !c.StderrRegex.MatchString(errStr) {
			t.Fatalf("stderr did not match regex for fixture %s.\nregex:  %v\nstderr: %s\n", c.BinFile, c.StderrRegex, errStr)
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
		is := afterDirTree.ToArray()
		shouldbe := expectedDirTree.ToArray()
		slices.Sort(is)
		slices.Sort(shouldbe)
		t.Fatalf("unexpected directory tree after Run().\nexpected: %v\ngot:      %v\n", shouldbe, is)
	}
}

func Test_Sqlite3DLL_Rename(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "./sqlite3win32x86",
			ExpectedFileName:   "./sqlite3win32x86/sqlite3.dll",
			StdoutRegex:        regexp.MustCompile(`.*Given/expected name: sqlite3win32x86 ↔ sqlite3.dll.*`),
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

	RunCasesAndCheck(t, cases, true, false, false)
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

	RunCasesAndCheck(t, cases, false, false, false)
}

func Test_Putty_Rename(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "./puttywin32x86",
			ExpectedFileName:   "./puttywin32x86/PuTTY.exe",
		},
		{
			BinFile:            "puttywin64x64",
			ObfuscatedFileName: "./puttywin64x64",
			ExpectedFileName:   "./puttywin64x64/PuTTY.exe",
		},
		{
			BinFile:            "puttywin64arm",
			ObfuscatedFileName: "./puttywin64arm",
			ExpectedFileName:   "./puttywin64arm/PuTTY.exe",
		},
	}

	RunCasesAndCheck(t, cases, false, false, false)
}

func Test_NSIS_Rename(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "NSISPortable311",
			ObfuscatedFileName: "./NSISPortable311",
			ExpectedFileName:   "./NSISPortable311/NSISPortable_3.11_English.paf.exe",
		},
	}

	RunCasesAndCheck(t, cases, false, false, false)
}

func Test_PNG_Rename(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "somepng",
			ObfuscatedFileName: "./somepng",
			ExpectedFileName:   "./somepng",
			StdoutRegex:        regexp.MustCompile(`.*File is not in PE format: DOS Header magic not found.*`),
		},
	}

	RunCasesAndCheck(t, cases, true, false, false)
}

func Test_Subfolder(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "somepng",
			ObfuscatedFileName: "./sub/abc",
			ExpectedFileName:   "./sub/abc",
		},
		{
			BinFile:            "somepng",
			ObfuscatedFileName: "./xyz",
			ExpectedFileName:   "./xyz",
		},
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "./sub/puttywin32x86",
			ExpectedFileName:   "./sub/puttywin32x86/PuTTY.exe",
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "./sqlite3win32x86",
			ExpectedFileName:   "./sqlite3win32x86/sqlite3.dll",
		},
	}

	RunCasesAndCheck(t, cases, false, false, false)
}

func Test_ExtEqualFlag(t *testing.T) {
	t.Skip("not yet - changed output")
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "puttywin32x86.exe",
			ExpectedFileName:   "puttywin32x86.exe/PuTTY.exe",
			StdoutRegex:        regexp.MustCompile(`.*extension matches: true.*`),
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "./sqlite3win32x86.dl_",
			ExpectedFileName:   "./sqlite3win32x86.dl_/sqlite3.dll",
			StdoutRegex:        regexp.MustCompile(`.*extension matches: false.*`),
		},
		{
			BinFile:            "NSISPortable311",
			ObfuscatedFileName: "./NSISPortable311",
			ExpectedFileName:   "./NSISPortable311/NSISPortable_3.11_English.paf.exe",
			StdoutRegex:        regexp.MustCompile(`.*extension matches: false.*`),
		},
	}

	RunCasesAndCheck(t, cases, true, false, false)
}

func Test_CorrectName(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "PuTTY.exe",
			ExpectedFileName:   "PuTTY.exe",
			StdoutRegex:        regexp.MustCompile(`.*Given/expected name: PuTTY.exe ↔ PuTTY.exe\n.*Similarity: 100\.0%.*`),
		},
	}

	RunCasesAndCheck(t, cases, true, false, false)
}

func Test_JustExt(t *testing.T) {
	cases := []testhelpers.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "puttywin32x86",
			ExpectedFileName:   "puttywin32x86.exe",
			StdoutRegex:        regexp.MustCompile(`(?s).*Given/expected name: puttywin32x86 ↔ puttywin32x86.exe.*Renaming .*puttywin32x86 → .*puttywin32x86.exe.*`),
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "sqlite3win32x86",
			ExpectedFileName:   "sqlite3win32x86.dll",
		},
	}

	RunCasesAndCheck(t, cases, true, false, true)
}
