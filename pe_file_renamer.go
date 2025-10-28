package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"os"

	levenshtein "github.com/TomTonic/levenshtein"
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

var candidates = make(map[string]RenamingCandidate, 0)

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

func SearchFiles(path string, verbose bool) error {

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
				err := SearchFiles(filepath.Join(path, e.Name()), verbose)
				if err != nil {
					return err
				}
			} else {
				processFile(filepath.Join(path, e.Name()), verbose)
			}
		}
	} else {
		processFile(path, verbose)
	}
	return nil
}

func processFile(filename string, verbose bool) {
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
			fmt.Printf("  Info: file is not in PE format: %v\n", err)
		}
		return
	}

	fileinfo := extractPEInfo(filename, pe)

	originalPath := filepath.Dir(filename)
	givenName := filepath.Base(filename)
	givenExt := filepath.Ext(filename)

	expectedName := fileinfo.Name
	expectedExt := filepath.Ext(expectedName)

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

func main() {
	verbose := flag.Bool("verbose", false, "include parse/open errors in output")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalf("Usage: %s [--verbose] <path-to-file-or-directory>", os.Args[0])
	}
	path := flag.Arg(0)

	err := SearchFiles(path, *verbose)
	if err != nil {
		log.Fatalf("Searching files: %v", err)
	}

	candidateList := make([]RenamingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateList = append(candidateList, candidate)
	}
	sortCandidates(candidateList)

	for _, candidate := range candidateList {
		fmt.Printf("# =========================================\n")
		fmt.Printf("# Original File Name: %s\n", candidate.OriginalName)
		fmt.Printf("# New Name:           %s\n", candidate.NewName)
		fmt.Printf("# Matching Ext:       %v\n", candidate.matching_extension)
		fmt.Printf("# Similarity:        %.1f%%\n", candidate.editing_distance_percentage)
		fmt.Printf("mv %s %s\n", strconv.Quote(filepath.Join(candidate.Path, candidate.OriginalName)), strconv.Quote(filepath.Join(candidate.Path, candidate.NewName)))
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
