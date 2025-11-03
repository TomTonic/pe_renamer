package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	misc "pe_renamer/misc"

	set3 "github.com/TomTonic/Set3"
	peparser "github.com/saferwall/pe"
)

func Test_FixtureObjectsPresentAndParseable(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	td := filepath.Join(repoRoot, "testdata")

	files := []string{
		"log4netdotnet20",
		"log4netdotnet462",
		"puttywin32x86",
		"puttywin64x64",
		"puttywin64arm",
		"sqlite3win32x86",
		"sqlite3win64x64",
		"sqlite3win64arm",
		"NSISPortable311",
	}

	for _, name := range files {
		path := filepath.Join(td, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("required fixture missing: %s", path)
		}
		p, err := peparser.New(path, &peparser.Options{})
		if err != nil {
			t.Fatalf("peparser.New(%s): %v", path, err)
		}
		if err := p.Parse(); err != nil {
			t.Fatalf("Parse failed for %s: %v", path, err)
		}
	}
}

// runCasesAndCheck runs Run() in-process against the provided FixtureCase slice.
// It returns stdout, stderr and the testdir (caller must cleanup).
func runCasesAndCheck(t *testing.T, cases []misc.FixtureObject, verbose bool, dryRun bool, justExt bool, ignoreCase bool) {
	//t.Helper()
	td := t.TempDir()
	defer func() { _ = os.RemoveAll(td) }()

	// prepare fixtures
	misc.CopyCasesToDir(t, cases, td)

	var stdout, stderr strings.Builder
	if err := Run(td, verbose, dryRun, justExt, ignoreCase, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v\nstderr: %s", err, stderr.String())
	}

	outStr := stdout.String()
	errStr := stderr.String()

	// capture directory tree before
	afterDirTree, err := misc.DirTree(t, td, ignoreCase)
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
			if ignoreCase {
				currentNormalized = strings.ToLower(currentNormalized)
			}
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
	cases := []misc.FixtureObject{
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "./sqlite3win32x86",
			ExpectedFileName:   "./sqlite3win32x86/sqlite3.dll",
			StdoutRegex:        regexp.MustCompile(`.*Expected name: sqlite3.dll.*`),
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

	runCasesAndCheck(t, cases, true, false, false, false)
}

func Test_Log4netDLL_Rename(t *testing.T) {
	cases := []misc.FixtureObject{
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

	runCasesAndCheck(t, cases, false, false, false, false)
}

func Test_Putty_Rename(t *testing.T) {
	cases := []misc.FixtureObject{
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

	runCasesAndCheck(t, cases, false, false, false, false)
}

func Test_NSIS_Rename(t *testing.T) {
	cases := []misc.FixtureObject{
		{
			BinFile:            "NSISPortable311",
			ObfuscatedFileName: "./NSISPortable311",
			ExpectedFileName:   "./NSISPortable311/NSISPortable_3.11_English.paf.exe",
		},
	}

	runCasesAndCheck(t, cases, false, false, false, false)
}

func Test_PNG_Rename(t *testing.T) {
	cases := []misc.FixtureObject{
		{
			BinFile:            "somepng",
			ObfuscatedFileName: "./somepng",
			ExpectedFileName:   "./somepng",
			StdoutRegex:        regexp.MustCompile(`.*File is not in PE format: DOS Header magic not found.*`),
		},
	}

	runCasesAndCheck(t, cases, true, false, false, false)
}

func Test_Subfolder(t *testing.T) {
	cases := []misc.FixtureObject{
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

	runCasesAndCheck(t, cases, false, false, false, false)
}

func Test_CorrectName(t *testing.T) {
	cases := []misc.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "PuTTY.exe",
			ExpectedFileName:   "PuTTY.exe",
			StdoutRegex:        regexp.MustCompile(`.*Expected name: PuTTY.exe\n.*Similarity: 100\.0%.*`),
		},
	}

	runCasesAndCheck(t, cases, true, false, false, false)
}

func Test_JustExt(t *testing.T) {
	cases := []misc.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "puttywin32x86",
			ExpectedFileName:   "puttywin32x86.exe",
			StdoutRegex:        regexp.MustCompile(`(?s).*Expected name: puttywin32x86.exe.*Renaming .*puttywin32x86 → .*puttywin32x86.exe.*`),
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "sqlite3win32x86",
			ExpectedFileName:   "sqlite3win32x86.dll",
		},
	}

	runCasesAndCheck(t, cases, true, false, true, false)
}

func Test_IgnoreCase(t *testing.T) {
	cases := []misc.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "putty.exe",
			ExpectedFileName:   "PuTTY.exe",
			StdoutRegex:        regexp.MustCompile(`(?s).*Expected name: PuTTY.exe.*`),
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "sqlite3.DLL",
			ExpectedFileName:   "SQLite3.dll",
		},
	}

	runCasesAndCheck(t, cases, true, false, false, true)
}

func Test_JustExtAndIgnoreCase(t *testing.T) {
	cases := []misc.FixtureObject{
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "putty.exe",
			ExpectedFileName:   "PuTTY.exe",
			StdoutRegex:        regexp.MustCompile(`(?s).*Expected name: putty.exe.*`),
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "sqlite3.DLL",
			ExpectedFileName:   "SQLite3.dll",
			StdoutRegex:        regexp.MustCompile(`(?s).*Expected name: sqlite3.dll.*`),
		},
		{
			BinFile:            "puttywin32x86",
			ObfuscatedFileName: "putty",
			ExpectedFileName:   "PuTTY.exe",
			StdoutRegex:        regexp.MustCompile(`(?s).*Expected name: putty.exe.*Renaming .*putty → .*putty.exe.*`),
		},
		{
			BinFile:            "sqlite3win32x86",
			ObfuscatedFileName: "sqlite3",
			ExpectedFileName:   "SQLite3.dll",
			StdoutRegex:        regexp.MustCompile(`(?s).*Expected name: sqlite3.dll.*Renaming .*sqlite3 → .*sqlite3.dll.*`),
		},
	}

	runCasesAndCheck(t, cases, true, false, true, true)
}

func Test_VersionOutput(t *testing.T) {
	// backup
	oldTag := gitTag
	oldOS := buildOS
	oldArch := buildArch
	defer func() {
		gitTag = oldTag
		buildOS = oldOS
		buildArch = oldArch
	}()

	gitTag = "v9.9.9-test"
	buildOS = "linux"
	buildArch = "amd64"

	var sb strings.Builder
	PrintVersion(&sb)
	out := sb.String()

	if !strings.Contains(out, "OS: linux") {
		t.Fatalf("expected OS in version output, got: %s", out)
	}
	if !strings.Contains(out, "ARCH: amd64") {
		t.Fatalf("expected ARCH in version output, got: %s", out)
	}
	if !strings.Contains(out, "TAG: v9.9.9-test") {
		t.Fatalf("expected TAG in version output, got: %s", out)
	}
}
