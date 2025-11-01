package testhelpers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	set3 "github.com/TomTonic/Set3"
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

// FixtureObject describes a single fixture mapping used in tests.
type FixtureObject struct {
	BinFile string
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

// CopyCasesToDir copies fixture directories for each case into dstDir, then
// moves one file from the copied fixture into the obfuscated path specified by
// the case. The obfuscated path is interpreted relative to dstDir.
func CopyCasesToDir(t *testing.T, cases []FixtureObject, dstDir string) {
	t.Helper()
	for _, c := range cases {
		CopyFromTestdata(t, c.BinFile, dstDir, c.ObfuscatedFileName)
	}
}

// CopyFixture copies a file from src to dst. It creates any necessary
// parent directories for dst.
func CopyFixture(t *testing.T, src, dst string) {
	t.Helper()

	dstPath := filepath.Dir(dst)
	infoDst, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dstPath, 0755); err != nil {
			t.Fatalf("CopyFixture mkdirall dst: %v", err)
		}
	} else if err != nil {
		t.Fatalf("CopyFixture stat dst: %v", err)
	} else if !infoDst.IsDir() {
		t.Fatalf("CopyFixture dst path is not a directory: %s", dstPath)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("CopyFixture open src %s: %v", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		t.Fatalf("CopyFixture create dst %s: %v", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("CopyFixture copy from %s to %s: %v", src, dst, err)
	}

}

// CopyFromTestdata copies one or more files or directories from the repository's `testdata`
// directory into dstDir. `sources` are file names within `testdata`. If dstName is non-empty,
// the copied file will be renamed to dstName inside dstDir.
func CopyFromTestdata(t *testing.T, source string, dstDir string, dstName string) {
	t.Helper()
	// find repository root (heuristic: directory containing go.mod)
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("CopyFromTestdata findRepoRoot: %v", err)
	}

	src := filepath.Join(repoRoot, "testdata", source)

	_, err = os.Stat(src)
	if err != nil {
		t.Fatalf("CopyFromTestdata stat %s: %v", src, err)
	}

	// if no specific destination name is given, keep the source base name
	if dstName == "" {
		dstName = source
	}

	dst := filepath.Join(dstDir, dstName)

	CopyFixture(t, src, dst)
}

// DirTree returns a set of file system paths (relative to root) for all files
// and directories under root. Each entry is prefixed with "D " for directories
// and "F " for files. Useful for asserting a created directory structure in tests.
func DirTree(t *testing.T, root string) (*set3.Set3[string], error) {
	t.Helper()
	out := set3.Empty[string]()
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
		entry := filepath.ToSlash(filepath.Clean(rel))
		if info.IsDir() {
			rel = "D " + entry
		} else {
			rel = "F " + entry
		}
		out.Add(rel)
		return nil
	})
	if err != nil {
		return set3.Empty[string](), err
	}
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
