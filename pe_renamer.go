package main

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"os"

	set3 "github.com/TomTonic/Set3"
	levenshtein "github.com/TomTonic/levenshtein"
	"github.com/alecthomas/kong"
	"github.com/google/uuid"
	peparser "github.com/saferwall/pe"
)

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

func extractPEInfo(path string, pe *peparser.File) FileInfo {
	name := "*"
	// 1) Prefer export name when available
	if pe.Export.Name != "" {
		name = pe.Export.Name
	} else {
		// 2) Then try CLR module/assembly name (if present)
		if modTable, ok := pe.CLR.MetadataTables[peparser.Module]; ok {
			if modTable.Content != nil {
				modTableRows := modTable.Content.([]peparser.ModuleTableRow)

				if len(modTableRows) > 0 {
					modName := pe.GetStringFromData(modTableRows[0].Name, pe.CLR.MetadataStreams["#Strings"])
					name = string(modName)
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
					break
				}
			}
		}
	}

	// 4) fallback to filename if no better name found
	if name == "*" {
		name = filepath.Base(path)
	}

	// ensure that the name has an appropriate extension
	ext := filepath.Ext(name)
	if ext == "" || !commonPEExtensions.Contains(strings.ToLower(ext)) {
		if pe.IsDLL() {
			name += ".dll"
		} else if pe.IsEXE() {
			name += ".exe"
		} else if pe.IsDriver() {
			name += ".sys"
		} else {
			name += ".bin"
		}
	}

	// extract version

	// Use export version if available, otherwise "*"
	version := "*"
	if pe.Export.Struct.MajorVersion != 0 || pe.Export.Struct.MinorVersion != 0 {
		version = fmt.Sprintf("%d.%d",
			pe.Export.Struct.MajorVersion,
			pe.Export.Struct.MinorVersion)
	} else {
		if pe.Resources.Struct.MajorVersion != 0 || pe.Resources.Struct.MinorVersion != 0 {
			version = fmt.Sprintf("%d.%d",
				pe.Resources.Struct.MajorVersion,
				pe.Resources.Struct.MinorVersion)
		} else {
			if asmTable, ok := pe.CLR.MetadataTables[peparser.Assembly]; ok {
				if asmTable.Content != nil {
					asmRows := asmTable.Content.([]peparser.AssemblyTableRow)
					if len(asmRows) > 0 {
						asm := asmRows[0]
						version = fmt.Sprintf("%d.%d.%d.%d", asm.MajorVersion, asm.MinorVersion, asm.BuildNumber, asm.RevisionNumber)
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
					break
				} else if pv, ok := kv["ProductVersion"]; ok && pv != "" {
					version = pv
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

func SearchFiles(path string, verbose bool, candidates *map[string]RenamingCandidate, out io.Writer, justExt bool) error {

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			fullChildName := filepath.Join(path, e.Name())
			if e.IsDir() {
				err := SearchFiles(fullChildName, verbose, candidates, out, justExt)
				if err != nil {
					return err
				}
			} else {
				processFile(fullChildName, verbose, candidates, out, justExt)
			}
		}
	} else {
		processFile(path, verbose, candidates, out, justExt)
	}
	return nil
}

func processFile(filename string, verbose bool, candidates *map[string]RenamingCandidate, out io.Writer, justExt bool) {
	if verbose {
		fmt.Fprintf(out, "File: %s\n", filename)
	}

	pe, err := peparser.New(filename, &peparser.Options{})
	if err != nil {
		if verbose {
			log.Printf("Error opening file %s: %v\n", filename, err)
		}
		return
	}
	// ensure any resources used by the parser are released (some backends keep files open)
	if closer, ok := any(pe).(interface{ Close() error }); ok {
		defer closer.Close()
	}

	if err := pe.Parse(); err != nil {
		if verbose {
			fmt.Fprintf(out, "  File is not in PE format: %v\n", err)
		}
		return
	}

	fileinfo := extractPEInfo(filename, pe)

	originalPath := filepath.Dir(filename)
	givenName := filepath.Base(filename)
	givenExt := filepath.Ext(filename)

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
		fmt.Fprintf(out, "  Given/expected name: %s ↔ %s\n", givenName, expectedName)
	}

	extEqual := false
	if strings.Compare(strings.ToUpper(givenExt), strings.ToUpper(expectedExt)) == 0 {
		extEqual = true
	}

	opts := levenshtein.Options{
		InsCost: 1, // if the filename is longer/more specific, don't penalize a lot. e.g. example.dll vs example32.dll and example64.dll
		DelCost: 10,
		SubCost: 20,
		Matches: func(a, b rune) bool {
			return unicode.ToLower(a) == unicode.ToLower(b)
		},
	}
	magicnumber := 2.5 // selected in a way that the resulting % values give a reasonnable representation of the actual equality

	distance := levenshtein.DistanceForStrings([]rune(expectedName), []rune(givenName), opts)
	sourceLength := float64(len(expectedName)) * magicnumber
	targetLength := float64(len(givenName)) * magicnumber
	equality := float64(sourceLength+targetLength-float64(distance)) / float64(sourceLength+targetLength)
	//equality := levenshtein.RatioForStrings([]rune(expectedName), []rune(givenName), opts)
	if equality < 0 {
		equality = 0
	}
	equality *= 100

	if verbose {
		fmt.Fprintf(out, "  Similarity: %.1f%%\n", equality)
	}

	if givenName == expectedName {
		return
	}

	candidate := RenamingCandidate{
		Path:                        originalPath,
		OriginalName:                givenName,
		NewName:                     expectedName,
		matching_extension:          extEqual,
		editing_distance_percentage: equality,
	}

	(*candidates)[filename] = candidate
}

// Run executes the main renaming-detection logic and writes human-readable
// operations to out (stdout) and logs to errWriter (stderr). It returns an error
// if searching or parsing fails.
func Run(out io.Writer, errWriter io.Writer, path string, verbose bool, dryRun bool, justExt bool) error {
	// set log output to errWriter so verbose parse errors are captured there
	log.SetOutput(errWriter)

	candidates := make(map[string]RenamingCandidate, 0)

	if err := SearchFiles(path, verbose, &candidates, out, justExt); err != nil {
		return err
	}

	candidateList := make([]RenamingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateList = append(candidateList, candidate)
	}
	sortCandidates(candidateList)

	for _, candidate := range candidateList {
		err := renameCandidate(out, candidate, verbose, dryRun)
		if err != nil {
			return err
		}
	}
	return nil
}

func renameCandidate(out io.Writer, candidate RenamingCandidate, verbose bool, dryRun bool) error {
	tempname := uuid.New().String()
	ofn := filepath.Join(candidate.Path, candidate.OriginalName)
	tmp := filepath.Join(candidate.Path, tempname)
	nfn := filepath.Join(ofn, candidate.NewName)

	if dryRun || verbose {
		// print the planned operation
		fmt.Fprintf(out, "Renaming %s → %s\n", ofn, nfn)
	}
	if dryRun {
		return nil
	}

	// perform the operations
	if err := os.Rename(ofn, tmp); err != nil {
		return err
	}
	if err := os.MkdirAll(ofn, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, nfn); err != nil {
		return err
	}
	return nil
}

func main() {
	var cli struct {
		Verbose    bool   `short:"v" help:"Print verbose output during processing"`
		DryRun     bool   `short:"n" help:"Don't perform writes; only show planned operations (dry-run)"`
		IgnoreCase bool   `short:"i" help:"Ignore case when comparing filenames"`
		JustExt    bool   `short:"e" help:"Only append an appropriate extension without renaming the base filename"`
		Path       string `arg:"" required:"" help:"Path to file or directory to search"`
	}

	ctx := kong.Parse(&cli, kong.Description("PE Renamer scans files or directories, identifies Windows PE files, and restores original filenames from embedded metadata. Improves compatibility with SBOM scanners and vulnerability tools like Syft and Grype.\n\nFor each renamed file the tool creates a directory named after the file's current name and moves the renamed file into that directory, so write permissions are required for the target location."))
	_ = ctx

	if err := Run(os.Stdout, os.Stderr, cli.Path, cli.Verbose, cli.DryRun, cli.JustExt); err != nil {
		log.Fatalf("run: %v", err)
	}
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
