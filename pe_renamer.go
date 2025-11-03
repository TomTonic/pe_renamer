package main

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"

	"os"

	misc "pe_renamer/misc"

	set3 "github.com/TomTonic/Set3"
	levenshtein "github.com/TomTonic/levenshtein"
	"github.com/alecthomas/kong"
	"github.com/google/uuid"
	peparser "github.com/saferwall/pe"
)

// (mustClose is defined later next to other helpers)

type FileInfo struct {
	Path    string
	Name    string
	Version string
}

type RenamingCandidate struct {
	Path                        string
	OriginalName                string
	NewName                     string
	matching_extension          bool
	editing_distance_percentage float64
}

var commonPEExtensions = set3.FromArray([]string{".exe", ".dll", ".sys", ".ocx", ".cpl", ".drv", ".scr"})

// build-time variables set via -ldflags. Defaults use runtime values.
var (
	gitTag    = "dev"
	buildOS   = runtime.GOOS
	buildArch = runtime.GOARCH
)

// PrintVersion writes OS/ARCH/TAG info to w.
func PrintVersion(w io.Writer) {
	_, _ = fmt.Fprintf(w, "OS: %s\n", buildOS)
	_, _ = fmt.Fprintf(w, "ARCH: %s\n", buildArch)
	_, _ = fmt.Fprintf(w, "TAG: %s\n", gitTag)
}

func init() {
	// Try to read build info embedded by the Go toolchain. In recent Go
	// versions the build may include VCS/module info (version or revision).
	// Only set gitTag from build info when it wasn't provided via ldflags
	// (the ldflags -X option overwrites the variable at link time). This
	// ensures an explicit ldflags value is preferred.
	if gitTag == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
			// Prefer a proper module version (starts with 'v') when present.
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" && strings.HasPrefix(bi.Main.Version, "v") {
				gitTag = bi.Main.Version
				return
			}
			// Otherwise look for VCS revision and use a short SHA as fallback.
			for _, s := range bi.Settings {
				if s.Key == "vcs.revision" && s.Value != "" {
					v := s.Value
					if len(v) > 12 {
						v = v[:12]
					}
					gitTag = v
					return
				}
			}
		}
	}

}

// mustClose closes the provided io.Closer and logs any error.
// Use this in defers to make intent explicit and surface closing errors.
// helper functions moved to package misc

func extractPEInfo(path string, pe *peparser.File, verbose bool, outWriter io.Writer) FileInfo {
	name := "*"
	// 1) Prefer export name when available
	if pe.Export.Name != "" {
		name = pe.Export.Name
		if verbose {
			_, _ = fmt.Fprintf(outWriter, "  Found name in export structure: %s\n", pe.Export.Name)
		}
	} else {
		// 2) Then try CLR module/assembly name (if present)
		if modTable, ok := pe.CLR.MetadataTables[peparser.Module]; ok {
			if modTable.Content != nil {
				modTableRows := modTable.Content.([]peparser.ModuleTableRow)

				if len(modTableRows) > 0 {
					modName := pe.GetStringFromData(modTableRows[0].Name, pe.CLR.MetadataStreams["#Strings"])
					name = string(modName)
					if verbose {
						_, _ = fmt.Fprintf(outWriter, "  Found CLR module/assembly name: %s\n", name)
					}
				}
			}
		}
	}

	// 3) STRINGFILEINFO
	if name == "*" {
		if sfi, err := pe.ParseVersionResourcesForEntries(); err == nil {
			// StringFileInfo is a map: langcodepage -> map[string]string
			for _, kv := range sfi {
				// kv is map[string]string
				if ofn, ok := kv["OriginalFilename"]; ok && ofn != "" {
					name = ofn
					if verbose {
						_, _ = fmt.Fprintf(outWriter, "  Found OriginalFilename in StringFileInfo: %s\n", name)
					}
					break
				}
			}
		}
	}

	// 4) fallback to filename if no better name found
	if name == "*" {
		name = filepath.Base(path)
		if verbose {
			_, _ = fmt.Fprintf(outWriter, "  Could not find better name, falling back to: %s\n", name)
		}
	}

	// ensure that the name has an appropriate extension
	ext := filepath.Ext(name)
	if ext == "" || !commonPEExtensions.Contains(strings.ToLower(ext)) {
		if verbose {
			_, _ = fmt.Fprintf(outWriter, "  Could not identify appropriate PE file extension. Guessing...\n")
		}
		if pe.IsDLL() {
			name += ".dll"
			if verbose {
				_, _ = fmt.Fprintf(outWriter, "  File seems to be a dynamic-link library (DLL). Appending extension: %s\n", name)
			}
		} else if pe.IsEXE() {
			name += ".exe"
			if verbose {
				_, _ = fmt.Fprintf(outWriter, "  File seems to be an executable (EXE). Appending extension: %s\n", name)
			}

		} else if pe.IsDriver() {
			name += ".sys"
			if verbose {
				_, _ = fmt.Fprintf(outWriter, "  File seems to be a driver (SYS). Appending extension: %s\n", name)
			}
		} else {
			name += ".bin"
			if verbose {
				_, _ = fmt.Fprintf(outWriter, "  Could not guess appropriate PE file extension. Using fallback: %s\n", name)
			}

		}
	}

	// extract version

	// Use export version if available, otherwise "*"
	version := "*"
	if pe.Export.Struct.MajorVersion != 0 || pe.Export.Struct.MinorVersion != 0 {
		version = fmt.Sprintf("%d.%d",
			pe.Export.Struct.MajorVersion,
			pe.Export.Struct.MinorVersion)
		if verbose {
			_, _ = fmt.Fprintf(outWriter, "  Found version information in export structure: %s\n", version)
		}
	} else {
		if pe.Resources.Struct.MajorVersion != 0 || pe.Resources.Struct.MinorVersion != 0 {
			version = fmt.Sprintf("%d.%d",
				pe.Resources.Struct.MajorVersion,
				pe.Resources.Struct.MinorVersion)
			if verbose {
				_, _ = fmt.Fprintf(outWriter, "  Found version information in resource structure: %s\n", version)
			}
		} else {
			if asmTable, ok := pe.CLR.MetadataTables[peparser.Assembly]; ok {
				if asmTable.Content != nil {
					asmRows := asmTable.Content.([]peparser.AssemblyTableRow)
					if len(asmRows) > 0 {
						asm := asmRows[0]
						version = fmt.Sprintf("%d.%d.%d.%d", asm.MajorVersion, asm.MinorVersion, asm.BuildNumber, asm.RevisionNumber)
						if verbose {
							_, _ = fmt.Fprintf(outWriter, "  Found version information in assembly table: %s\n", version)
						}
					}
				}
			}
		}
	}

	// 3) STRINGFILEINFO
	if version == "*" {
		if sfi, err := pe.ParseVersionResourcesForEntries(); err == nil {
			// StringFileInfo is a map: langcodepage -> map[string]string
			for _, kv := range sfi {
				if fv, ok := kv["FileVersion"]; ok && fv != "" {
					version = fv
					if verbose {
						_, _ = fmt.Fprintf(outWriter, "  Found version information in StringFileInfo (FileVersion): %s\n", version)
					}
					break
				} else if pv, ok := kv["ProductVersion"]; ok && pv != "" {
					version = pv
					if verbose {
						_, _ = fmt.Fprintf(outWriter, "  Found version information in StringFileInfo (ProductVersion): %s\n", version)
					}
					break
				}
			}
		}
	}

	return FileInfo{
		Path:    path,
		Name:    name,
		Version: version,
	}
}

func searchFiles(path string, verbose bool, dryRun bool, justExt bool, ignoreCase bool, candidates *map[string]RenamingCandidate, outWriter io.Writer, errWriter io.Writer) error {

	info, err := os.Stat(path)
	if err != nil {
		_, _ = fmt.Fprintf(errWriter, "Error getting information about %q: %s\n", path, misc.ConciseErr(err))
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			_, _ = fmt.Fprintf(errWriter, "Error reading directory %q: %s\n", path, misc.ConciseErr(err))
			return err
		}
		for _, e := range entries {
			fullChildName := filepath.Join(path, e.Name())
			if e.IsDir() {
				if err := searchFiles(fullChildName, verbose, dryRun, justExt, ignoreCase, candidates, outWriter, errWriter); err != nil {
					return err
				}
			} else {
				processFile(fullChildName, verbose, dryRun, justExt, ignoreCase, candidates, outWriter, errWriter)
			}
		}
	} else {
		processFile(path, verbose, dryRun, justExt, ignoreCase, candidates, outWriter, errWriter)
	}
	return nil

}

func processFile(path string, verbose bool, dryRun bool, justExt bool, ignoreCase bool, candidates *map[string]RenamingCandidate, outWriter io.Writer, errWriter io.Writer) {
	if verbose {
		_, _ = fmt.Fprintf(outWriter, "File: %s\n", path)
	}

	pe, err := peparser.New(path, &peparser.Options{})
	if err != nil {
		_, _ = fmt.Fprintf(errWriter, "Error opening file %q: %s\n", path, misc.ConciseErr(err))
		return
	}
	// ensure any resources used by the parser are released (some backends keep files open)
	if closer, ok := any(pe).(interface{ Close() error }); ok {
		defer misc.MustClose(closer, errWriter)
	}

	if err := pe.Parse(); err != nil {
		if verbose {
			_, _ = fmt.Fprintf(outWriter, "  File is not in PE format: %v\n", err)
		}
		return
	}
	fileinfo := extractPEInfo(path, pe, verbose, outWriter)

	originalPath := filepath.Dir(path)
	givenName := filepath.Base(path)
	givenExt := filepath.Ext(path)

	expectedName := fileinfo.Name
	expectedExt := filepath.Ext(expectedName)
	expectedNameWithoutExt := strings.TrimSuffix(expectedName, expectedExt)
	expectedExt = strings.ToLower(expectedExt)
	expectedName = expectedNameWithoutExt + expectedExt

	// If JustExt flag is set, only change the extension; keep original base name
	if justExt {
		base := strings.TrimSuffix(givenName, givenExt)
		expectedName = base + expectedExt
	}

	if verbose {
		_, _ = fmt.Fprintf(outWriter, "  Expected name: %s\n", expectedName)
	}

	// prefer a direct case-insensitive equality check
	extEqual := strings.EqualFold(givenExt, expectedExt)

	opts := levenshtein.Options{
		InsCost: 1,
		DelCost: 1,
		SubCost: 2,
		Matches: func(a, b rune) bool {
			result := (ignoreCase && (unicode.ToLower(a) == unicode.ToLower(b))) || (!ignoreCase && (a == b))
			return result
		},
	}
	equality := levenshtein.RatioForStrings([]rune(expectedName), []rune(givenName), opts)
	equality *= 100

	if verbose {
		_, _ = fmt.Fprintf(outWriter, "  Similarity: %.1f%%\n", equality)
	}

	// consider ignoreCase when determining if names are already equal
	if (ignoreCase && strings.EqualFold(givenName, expectedName)) || (!ignoreCase && givenName == expectedName) {
		if verbose {
			if ignoreCase {
				_, _ = fmt.Fprintf(outWriter, "  Regarding file names as equal (ignore case). Skipping rename.\n")
			} else {
				_, _ = fmt.Fprintf(outWriter, "  Regarding file names as equal. Skipping rename.\n")
			}
		}
		return
	}

	candidate := RenamingCandidate{
		Path:                        originalPath,
		OriginalName:                givenName,
		NewName:                     expectedName,
		matching_extension:          extEqual,
		editing_distance_percentage: equality,
	}

	(*candidates)[path] = candidate
}

// Run executes the main renaming-detection logic and writes human-readable
// operations to out (stdout) and logs to errWriter (stderr).
func Run(path string, verbose bool, dryRun bool, justExt bool, ignoreCase bool, out io.Writer, errWriter io.Writer) error {
	// Note: explicit logging calls were replaced by writes to errWriter.

	candidates := make(map[string]RenamingCandidate, 0)

	if err := searchFiles(path, verbose, dryRun, justExt, ignoreCase, &candidates, out, errWriter); err != nil {
		return err
	}

	candidateList := make([]RenamingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateList = append(candidateList, candidate)
	}
	sortCandidates(candidateList)

	for _, candidate := range candidateList {
		renameCandidate(candidate, verbose, dryRun, justExt, ignoreCase, out, errWriter)
	}
	return nil
}

func renameCandidate(candidate RenamingCandidate, verbose bool, dryRun bool, justExt bool, ignoreCase bool, outWriter io.Writer, errWriter io.Writer) {
	tempname := uuid.New().String()
	ofn := filepath.Join(candidate.Path, candidate.OriginalName)
	tmp := filepath.Join(candidate.Path, tempname)
	nfn := filepath.Join(ofn, candidate.NewName)

	if justExt {
		nfn = filepath.Join(candidate.Path, candidate.NewName)
	}

	if dryRun || verbose {
		// print the planned operation
		_, _ = fmt.Fprintf(outWriter, "Renaming %s → %s\n", ofn, nfn)
	}
	if dryRun {
		return
	}

	if justExt {
		// perform simple rename
		if err := os.Rename(ofn, nfn); err != nil {
			_, _ = fmt.Fprintf(errWriter, "Error renaming %q to %q: %s\n", ofn, nfn, misc.ConciseErr(err))
			return
		}
		return
	}

	// perform complex rename: move original file to temp name, create dir with original name, move temp file into that dir with new name
	if err := os.Rename(ofn, tmp); err != nil {
		_, _ = fmt.Fprintf(errWriter, "Error renaming %q to temporary file %q: %s\n", ofn, tmp, misc.ConciseErr(err))
		return
	}
	if err := os.MkdirAll(ofn, 0o755); err != nil {
		_, _ = fmt.Fprintf(errWriter, "Error creating directory %q: %s\n", ofn, misc.ConciseErr(err))
		return
	}
	if err := os.Rename(tmp, nfn); err != nil {
		_, _ = fmt.Fprintf(errWriter, "Error renaming temporary file %q to %q: %s\n", tmp, nfn, misc.ConciseErr(err))
		return
	}
}

func main() {
	os.Exit(runCli(os.Args[1:], os.Stdout, os.Stderr))
}

// runCli executes the CLI logic for the given args and writes to the provided
// stdout/stderr. It returns an exit code that can be passed to os.Exit.
func runCli(args []string, stdout, stderr io.Writer) int {
	// prepare CLI struct similar to main
	var cli struct {
		Version    bool   `help:"Print version and exit" name:"version" short:"V"`
		Verbose    bool   `short:"v" help:"Print verbose output during processing"`
		DryRun     bool   `short:"n" help:"Don't perform writes; only show planned operations (dry-run)"`
		IgnoreCase bool   `short:"i" help:"Ignore case when comparing filenames"`
		JustExt    bool   `short:"e" help:"Only append an appropriate extension without renaming the base filename"`
		Path       string `arg:"" optional:"" help:"Path to file or directory to search"`
	}

	// Temporarily set os.Args so kong parses the provided args slice.
	savedArgs := os.Args
	defer func() { os.Args = savedArgs }()
	if len(savedArgs) > 0 {
		os.Args = append([]string{savedArgs[0]}, args...)
	} else {
		os.Args = append([]string{"pe_renamer"}, args...)
	}

	// Parse using kong but direct output to provided writers.
	ctx := kong.Parse(&cli,
		kong.Description("PE Renamer scans files or directories, identifies Windows PE files, and restores original filenames from embedded metadata. Improves compatibility with SBOM scanners and vulnerability tools like Syft and Grype."),
		kong.Writers(stdout, stderr),
	)
	_ = ctx

	if cli.Version {
		PrintVersion(stdout)
		return 0
	}

	if cli.Path == "" {
		_, _ = fmt.Fprintln(stderr, "path is required (use -h for help or --version to show build info)")
		return 2
	}

	if err := Run(cli.Path, cli.Verbose, cli.DryRun, cli.JustExt, cli.IgnoreCase, stdout, stderr); err != nil {
		return 1
	}
	return 0
}

func sortCandidates(candidates []RenamingCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		// 1. Zuerst die mit gleicher Extension
		if candidates[i].matching_extension != candidates[j].matching_extension {
			return candidates[i].matching_extension
		}
		// 2. Nach editing_distance_percentage absteigend
		if candidates[i].editing_distance_percentage != candidates[j].editing_distance_percentage {
			return candidates[i].editing_distance_percentage > candidates[j].editing_distance_percentage
		}
		// 3. Nach Pfad aufsteigend
		if candidates[i].Path != candidates[j].Path {
			return candidates[i].Path < candidates[j].Path
		}
		// 4. Nach OriginalName aufsteigend
		return candidates[i].OriginalName < candidates[j].OriginalName
	})
}
