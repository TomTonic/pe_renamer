package main

import (
	"os"
	"path/filepath"
	"testing"

	peparser "github.com/saferwall/pe"
)

// check_fixtures_test expects the following files to be present under testdata/
//
//   - log4netdotnet20
//   - log4netdotnet462
//   - puttywin32x86
//   - puttywin64x64
//   - puttywin64arm
//   - sqlite3win32x86
//   - sqlite3win64x64
//   - sqlite3win64arm
//
// The test fails if any file is missing or if the file cannot be parsed as PE.
func TestFixtureObjectsPresentAndParseable(t *testing.T) {
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
