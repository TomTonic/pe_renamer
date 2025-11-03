# PE Renamer

[![Go Report Card](https://goreportcard.com/badge/github.com/TomTonic/pe_renamer)](https://goreportcard.com/report/github.com/TomTonic/pe_renamer)

[![golangci-lint](https://github.com/TomTonic/pe_renamer/actions/workflows/ci.yml/badge.svg)](https://github.com/TomTonic/pe_renamer/actions/workflows/ci.yml)

[![CodeQL](https://github.com/TomTonic/pe_renamer/actions/workflows/codeql.yml/badge.svg)](https://github.com/TomTonic/pe_renamer/actions/workflows/codeql.yml)

[![Tests](https://github.com/TomTonic/pe_renamer/actions/workflows/coverage.yml/badge.svg?branch=main)](https://github.com/TomTonic/pe_renamer/actions/workflows/coverage.yml)

![coverage](https://raw.githubusercontent.com/TomTonic/pe_renamer/badges/.badges/main/coverage.svg)

PE Renamer is a command-line tool designed to scan files or directories, identify Windows Portable Executable (PE) files, and restore their original filenames based on embedded metadata. This helps improve compatibility with SBOM scanners and vulnerability tools like Syft and Grype.

## What is a PE file?

PE stands for **Portable Executable**, the file format used by Windows for executables (`.exe`), dynamic link libraries (`.dll`), system drivers (`.sys`), and more. It is based on the COFF (Common Object File Format) and includes headers, sections, and metadata that describe how the operating system should load and execute the file.

## Features

- Recursively scans a file or directory
- Attempts to interpret each file as a PE file (regardless of extension or name)
- Extracts the original filename from PE metadata (if available)
- Creates a directory named after the original filename
- Moves and renames the PE file into the new directory
- Platform independent, the tool runs on Windows, Linux, MacOS, etc.

## Use Case

Many installation archives or package bundles contain PE files with obfuscated or generic names. This can hinder tools like:

- [Syft](https://github.com/anchore/syft) – for generating Software Bill of Materials (SBOM)
- [Grype](https://github.com/anchore/grype) – for vulnerability scanning

By restoring original filenames and organizing files into meaningful directories, PE Renamer improves the accuracy and effectiveness of these tools.

## Example

Suppose you have a directory with files like `_CBA1F54FF12A5D6D107C97BFFEFC2C62`, `Global_VC_ATLANSI_f0.7EBEDD68_AA66_11D2_B980_006097C4DE24`, etc. PE Renamer will:

1. Identify which of these are valid PE files.
2. Extract the original filename from the PE metadata (e.g., `log4net.dll` resp. `ATL.dll`).
3. Create a folders named `_CBA1F54FF12A5D6D107C97BFFEFC2C62/` resp. `Global_VC_ATLANSI_f0.7EBEDD68_AA66_11D2_B980_006097C4DE24` and move the files inside, renaming it to `log4net.dll` resp. `ATL.dll`.

## Installation

From source (recommended)

```bash
# installs into $(go env GOPATH)/bin (Go 1.17+)
go install github.com/TomTonic/pe_renamer@latest
```

Make sure your `$GOPATH/bin` (or `$(go env GOPATH)/bin`) is in your PATH.

Build locally

```bash
git clone https://github.com/TomTonic/pe_renamer.git
cd pe_renamer
go build -o pe_renamer ./...
# Windows:
# go build -o pe_renamer.exe ./...
```

## Usage

### Basic

Running without arguments prints a short usage error message:

```text
code/pe_renamer$ ./pe_renamer
path is required (use -h for help or --version to show build info)
```

The CLI supports a short help message:

```text
code/pe_renamer$ ./pe_renamer -h            
Usage: pe_renamer [<path>] [flags]

PE Renamer scans files or directories, identifies Windows PE files, and restores original filenames from embedded metadata. Improves compatibility with SBOM scanners and vulnerability tools like Syft and Grype.

Arguments:
	[<path>]    Path to file or directory to search

Flags:
	-h, --help           Show context-sensitive help.
	-V, --version        Print version and exit
	-v, --verbose        Print verbose output during processing
	-n, --dry-run        Don't perform writes; only show planned operations (dry-run)
	-i, --ignore-case    Ignore case when comparing filenames
	-e, --just-ext       Only append an appropriate extension without renaming the base filename
```

### Example

Dry-run with verbose output, case-insensitive (sample):

```text
code/pe_renamer$ ./pe_renamer -v -i --dry-run ./testdata
File: testdata/NSISPortable311
  Found OriginalFilename in StringFileInfo: NSISPortable_3.11_English.paf.exe
  Found version information in StringFileInfo (FileVersion): 3.11.0.0
  Expected name: NSISPortable_3.11_English.paf.exe
  Similarity: 62.5%
File: testdata/README.md
  File is not in PE format: DOS Header magic not found
File: testdata/log4netdotnet20
  Found CLR module/assembly name: log4net.dll
  Found version information in assembly table: 3.2.0.0
  Expected name: log4net.dll
  Similarity: 61.5%
File: testdata/log4netdotnet462
  Found CLR module/assembly name: log4net.dll
  Found version information in assembly table: 3.2.0.0
  Expected name: log4net.dll
  Similarity: 59.3%
File: testdata/puttywin32x86
  Found OriginalFilename in StringFileInfo: PuTTY
  Could not identify appropriate PE file extension. Guessing...
  File seems to be an executable (EXE). Appending extension: PuTTY.exe
  Found version information in StringFileInfo (FileVersion): Release 0.83 (with embedded help)
  Expected name: PuTTY.exe
  Similarity: 54.5%
File: testdata/puttywin64arm
  Found OriginalFilename in StringFileInfo: PuTTY
  Could not identify appropriate PE file extension. Guessing...
  File seems to be an executable (EXE). Appending extension: PuTTY.exe
  Found version information in StringFileInfo (FileVersion): Release 0.83 (with embedded help)
  Expected name: PuTTY.exe
  Similarity: 45.5%
File: testdata/puttywin64x64
  Found OriginalFilename in StringFileInfo: PuTTY
  Could not identify appropriate PE file extension. Guessing...
  File seems to be an executable (EXE). Appending extension: PuTTY.exe
  Found version information in StringFileInfo (FileVersion): Release 0.83 (with embedded help)
  Expected name: PuTTY.exe
  Similarity: 54.5%
File: testdata/somepng
  File is not in PE format: DOS Header magic not found
File: testdata/sqlite3win32x86
  Found name in export structure: sqlite3.dll
  Found version information in StringFileInfo (FileVersion): 3.50.4
  Expected name: sqlite3.dll
  Similarity: 53.8%
File: testdata/sqlite3win64arm
  Found name in export structure: sqlite3.dll
  Found version information in StringFileInfo (FileVersion): 3.50.4
  Expected name: sqlite3.dll
  Similarity: 53.8%
File: testdata/sqlite3win64x64
  Found name in export structure: sqlite3.dll
  Found version information in StringFileInfo (FileVersion): 3.50.4
  Expected name: sqlite3.dll
  Similarity: 53.8%
Renaming testdata/NSISPortable311 → testdata/NSISPortable311/NSISPortable_3.11_English.paf.exe
Renaming testdata/log4netdotnet20 → testdata/log4netdotnet20/log4net.dll
Renaming testdata/log4netdotnet462 → testdata/log4netdotnet462/log4net.dll
Renaming testdata/puttywin32x86 → testdata/puttywin32x86/PuTTY.exe
Renaming testdata/puttywin64x64 → testdata/puttywin64x64/PuTTY.exe
Renaming testdata/sqlite3win32x86 → testdata/sqlite3win32x86/sqlite3.dll
Renaming testdata/sqlite3win64arm → testdata/sqlite3win64arm/sqlite3.dll
Renaming testdata/sqlite3win64x64 → testdata/sqlite3win64x64/sqlite3.dll
Renaming testdata/puttywin64arm → testdata/puttywin64arm/PuTTY.exe
```

### Notes

- Use --dry-run to verify the effects before applying them.
- The tool attempts multiple heuristics (exports, CLR metadata, PE version resources) to derive an original filename; it may not always find a perfect name.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
