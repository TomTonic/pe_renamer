package main

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"os"

	levenshtein "github.com/TomTonic/levenshtein"
	"github.com/alecthomas/kong"
	"github.com/google/uuid"
	peparser "github.com/saferwall/pe"
)

type FileInfo struct {
	Path       string
	Name       string
	Version    string
	hasExports bool
}

type RenamingCandidate struct {
	Path                        string
	OriginalName                string
	NewName                     string
	matching_extension          bool
	editing_distance_percentage float64
}

func extractPEInfo(path string, pe *peparser.File) FileInfo {
	name := filepath.Base(path)
	if pe.Export.Name != "" {
		name = pe.Export.Name
	} else {
		if modTable, ok := pe.CLR.MetadataTables[peparser.Module]; ok {
			if modTable.Content != nil {
				modTableRows := modTable.Content.([]peparser.ModuleTableRow)

				if len(modTableRows) > 0 {
					modName := pe.GetStringFromData(modTableRows[0].Name, pe.CLR.MetadataStreams["#Strings"])
					name = string(modName)
					// log.Println("Assembly-/Modulname:", name)
				} else {
					// log.Println("Keine ModuleTableRows gefunden.")
				}
			}
		}
	}

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
						// log.Println("Assembly-Version:", version)
					}
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

func SearchFiles(path string, verbose bool, candidates map[string]RenamingCandidate) error {

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
			if e.IsDir() {
				err := SearchFiles(filepath.Join(path, e.Name()), verbose, candidates)
				if err != nil {
					return err
				}
			} else {
				processFile(filepath.Join(path, e.Name()), verbose, candidates)
			}
		}
	} else {
		processFile(path, verbose, candidates)
	}
	return nil
}

func processFile(filename string, verbose bool, candidates map[string]RenamingCandidate) {
	if verbose {
		log.Printf("File: %s\n", filename)
	}

	pe, err := peparser.New(filename, &peparser.Options{})
	if err != nil {
		if verbose {
			log.Printf("  Error opening file: %v\n", err)
		}
		return
	}

	if err := pe.Parse(); err != nil {
		if verbose {
			log.Printf("  Info: file is not in PE format: %v\n", err)
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

	if givenName == expectedName {
		return
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

	candidate := RenamingCandidate{
		Path:                        originalPath,
		OriginalName:                givenName,
		NewName:                     expectedName,
		matching_extension:          extEqual,
		editing_distance_percentage: equality * 100,
	}

	candidates[filename] = candidate
}

// Run executes the main renaming-detection logic and writes human-readable
// operations to out (stdout) and logs to errWriter (stderr). It returns an error
// if searching or parsing fails.
func Run(path string, verbose bool, out io.Writer, errWriter io.Writer) error {
	// set log output to errWriter so verbose parse errors are captured there
	log.SetOutput(errWriter)

	candidates := make(map[string]RenamingCandidate, 0)

	if err := SearchFiles(path, verbose, candidates); err != nil {
		return err
	}

	candidateList := make([]RenamingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateList = append(candidateList, candidate)
	}
	sortCandidates(candidateList)

	for _, candidate := range candidateList {
		tempname := uuid.New().String()
		fmt.Fprintf(out, "# =========================================\n")
		fmt.Fprintf(out, "# Original File Name: %s\n", candidate.OriginalName)
		fmt.Fprintf(out, "# New Name:           %s\n", candidate.NewName)
		fmt.Fprintf(out, "# Matching Ext:       %v\n", candidate.matching_extension)
		fmt.Fprintf(out, "# Similarity:        %.1f%%\n", candidate.editing_distance_percentage)
		fmt.Fprintf(out, "mv %s %s\n", strconv.Quote(filepath.Join(candidate.Path, candidate.OriginalName)), strconv.Quote(filepath.Join(candidate.Path, tempname)))
		fmt.Fprintf(out, "mkdir %s\n", strconv.Quote(filepath.Join(candidate.Path, candidate.OriginalName)))
		fmt.Fprintf(out, "mv %s %s\n", strconv.Quote(filepath.Join(candidate.Path, tempname)), strconv.Quote(filepath.Join(candidate.Path, candidate.OriginalName, candidate.NewName)))
	}
	return nil
}

func main() {
	var cli struct {
		Verbose bool   `short:"v" help:"Include parse/open errors in output"`
		Path    string `arg:"" required:"" help:"Path to file or directory to search"`
	}

	ctx := kong.Parse(&cli, kong.Description("PE Renamer scans files or directories, identifies Windows PE files, and restores original filenames from embedded metadata. Improves compatibility with SBOM scanners and vulnerability tools like Syft and Grype.\n\nFor each renamed file the tool creates a directory named after the file's current name and moves the renamed file into that directory, so write permissions are required for the target location."))
	_ = ctx

	if err := Run(cli.Path, cli.Verbose, os.Stdout, os.Stderr); err != nil {
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
