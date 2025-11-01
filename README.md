# PE Renamer

[![Go Report Card](https://goreportcard.com/badge/github.com/TomTonic/pe_renamer)](https://goreportcard.com/report/github.com/TomTonic/pe_renamer)
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

```bash
pe_renamer [flags] <path>
```

### Common flags

- --dry-run, -n    : don't perform filesystem changes, only print planned renaming operations
- --verbose, -v    : print verbose output during processing
- --ignoreCase, -i : perform filename comparisons case-insensitively. When set, name-equality checks use case-insensitive matching (useful on Windows-like collated filesystems).
- --justExt, -e    : only adjust/append the file extension; do not change the base filename. Useful when you only need to restore an extension (e.g. add `.dll` to obfuscated binaries) without renaming the file's base name.

### Examples

Dry-run (inspect planned changes)

```bash
# Linux/macOS
pe_renamer --dry-run --verbose ./packages

# Windows PowerShell
.\pe_renamer.exe -n --verbose .\packages
```

Apply changes (actually perform renames)

```bash
pe_renamer ./packages
```

### Notes

- Use --dry-run to verify the effects before applying them.
- The tool attempts multiple heuristics (exports, CLR metadata, PE version resources) to derive an original filename; it may not always find a perfect name.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
