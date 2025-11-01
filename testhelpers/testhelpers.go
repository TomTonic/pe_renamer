package testhelpers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// findRepoRoot searches parent directories for a go.mod file and returns the directory
// containing it. This helps resolving paths relative to the repository root.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// FixtureCase describes a single fixture mapping used in tests.
type FixtureCase struct {
	Fixture string
	// ObfuscatedFileName is the relative path (within the test dir) where the
	// original file will be moved to simulate an obfuscated/unknown filename.
	ObfuscatedFileName string
	// ExpectedFileName is the relative path (within the test dir) expected after
	// the tool's operations have been applied (e.g. "./abc/log4net.dll").
	ExpectedFileName string
	// Optional regexes: if non-nil, they must match the overall stdout/stderr.
	StdoutRegex *regexp.Regexp
	StderrRegex *regexp.Regexp
}

// StdoutRegex matches Run() stdout lines (summary headers, mv/mkdir commands).
var StdoutRegex = regexp.MustCompile(`(?m)^(#\s+Original File Name:|#\s+New Name:|mv\s|mkdir\s)`)

// StderrRegex matches common Run() stderr log lines (e.g. parse/open file messages).
var StderrRegex = regexp.MustCompile(`(?m)File:\s+`)

// CopyCasesToDir copies fixture directories for each case into dstDir, then
// moves one file from the copied fixture into the obfuscated path specified by
// the case. The obfuscated path is interpreted relative to dstDir.
func CopyCasesToDir(t *testing.T, cases []FixtureCase, dstDir string) {
	t.Helper()
	for _, c := range cases {
		CopyFromTestdata(t, []string{c.Fixture}, dstDir, nil)

		// Determine where the copied content landed. It may be a directory
		// (dstDir/<fixture>) or a single file (dstDir/<basename>). Handle both.
		srcDir := filepath.Join(dstDir, c.Fixture)
		var srcFile string
		if fi, err := os.Stat(srcDir); err == nil && fi.IsDir() {
			entries, err := os.ReadDir(srcDir)
			if err != nil {
				t.Fatalf("read copied fixture dir: %v", err)
			}
			if len(entries) == 0 {
				t.Fatalf("fixture dir %s is empty", srcDir)
			}
			for _, e := range entries {
				if !e.IsDir() {
					srcFile = filepath.Join(srcDir, e.Name())
					break
				}
			}
			if srcFile == "" {
				srcFile = filepath.Join(srcDir, entries[0].Name())
			}
		} else {
			// maybe CopyFromTestdata copied a single file into dstDir
			base := filepath.Base(c.Fixture)
			candidate := filepath.Join(dstDir, base)
			if fi2, err2 := os.Stat(candidate); err2 == nil && !fi2.IsDir() {
				srcFile = candidate
			} else {
				// fallback: find any non-dir entry directly in dstDir
				entries, err := os.ReadDir(dstDir)
				if err != nil {
					t.Fatalf("read dstDir to find copied fixture: %v", err)
				}
				for _, e := range entries {
					if !e.IsDir() {
						srcFile = filepath.Join(dstDir, e.Name())
						break
					}
				}
			}
			if srcFile == "" {
				t.Fatalf("could not locate copied fixture for %s in %s", c.Fixture, dstDir)
			}
		}

		dest := filepath.Join(dstDir, strings.TrimPrefix(c.ObfuscatedFileName, "./"))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatalf("mkdirall for obf dest: %v", err)
		}
		if err := os.Rename(srcFile, dest); err != nil {
			t.Fatalf("moving fixture file to obf path failed: %v", err)
		}
		// remove the now-empty fixture dir if it exists
		_ = os.RemoveAll(srcDir)
	}
}

// CopyFixture copies a file or directory from testdata to dstPath. If dstName is non-empty
// the copied file will be renamed to dstName inside dstPath. If srcPath is a directory, it
// copies the directory contents recursively.
func CopyFixture(t *testing.T, srcPath, dstPath, dstName string) {
	t.Helper()

	info, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("CopyFixture stat src: %v", err)
	}

	if !info.IsDir() {
		// srcPath is file
		srcFile, err := os.Open(srcPath)
		if err != nil {
			t.Fatalf("CopyFixture open src: %v", err)
		}
		defer srcFile.Close()

		name := info.Name()
		if dstName != "" {
			name = dstName
		}
		dstFilePath := filepath.Join(dstPath, name)
		dstFile, err := os.Create(dstFilePath)
		if err != nil {
			t.Fatalf("CopyFixture create dst: %v", err)
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			t.Fatalf("CopyFixture copy: %v", err)
		}
		return
	}

	// srcPath is a directory; copy recursively
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		t.Fatalf("CopyFixture readdir: %v", err)
	}

	for _, e := range entries {
		srcEntryPath := filepath.Join(srcPath, e.Name())
		dstEntryPath := filepath.Join(dstPath, e.Name())
		if e.IsDir() {
			if err := copyDir(srcEntryPath, dstEntryPath); err != nil {
				t.Fatalf("CopyFixture copyDir: %v", err)
			}
		} else {
			CopyFixture(t, srcEntryPath, dstPath, "")
		}
	}
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcEntry := filepath.Join(src, e.Name())
		dstEntry := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcEntry, dstEntry); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcEntry, dstEntry); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()
	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstF.Close()
	_, err = io.Copy(dstF, srcF)
	return err
}

// CopyFromTestdata copies one or more files or directories from the repository's `testdata`
// directory into dstDir. `sources` are paths relative to the repository root (they may
// start with or without `testdata/`). If `renames` contains a key matching the base name
// of a source, that entry will be used as the destination name for that source.
func CopyFromTestdata(t *testing.T, sources []string, dstDir string, renames map[string]string) {
	t.Helper()
	// find repository root (heuristic: directory containing go.mod)
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("CopyFromTestdata findRepoRoot: %v", err)
	}

	for _, srcRel := range sources {
		// allow the caller to pass either "testdata/xxx" or just "xxx"
		if srcRel == "" {
			continue
		}
		src := srcRel
		if !filepath.IsAbs(src) {
			// if srcRel doesn't already start with testdata, prefix it
			if !(len(srcRel) >= 8 && srcRel[:8] == "testdata") {
				src = filepath.Join("testdata", srcRel)
			}
			src = filepath.Join(repoRoot, src)
		}

		info, err := os.Stat(src)
		if err != nil {
			t.Fatalf("CopyFromTestdata stat %s: %v", src, err)
		}

		if info.IsDir() {
			// copy directory contents into dstDir/<dirbasename>
			d := filepath.Join(dstDir, info.Name())
			if err := copyDir(src, d); err != nil {
				t.Fatalf("CopyFromTestdata copyDir: %v", err)
			}
			continue
		}

		// file
		base := info.Name()
		dstName := base
		if v, ok := renames[base]; ok && v != "" {
			dstName = v
		}
		CopyFixture(t, src, dstDir, dstName)
	}
}

// DirTree returns a sorted list of file system paths (relative to root) for all files
// and directories under root. Useful for asserting a created directory structure in tests.
func DirTree(t *testing.T, root string) ([]string, error) {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	// sort for deterministic order
	sort.Strings(out)
	return out, nil
}

// CreateTestDir creates a new temporary directory for tests and returns its path.
// Caller is responsible for cleanup (os.RemoveAll).
func CreateTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pe_renamer_test_")
	if err != nil {
		t.Fatalf("CreateTestDir: %v", err)
	}
	return dir
}

// FileSHA256 returns the SHA256 checksum of the file at path as a hex string.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
