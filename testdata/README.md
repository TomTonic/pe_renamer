# testdata: downloaded binary fixtures

This directory contains binary test fixtures used by the tests in this repository. These binaries are only used as test fixtures. The repository does not redistribute them.

Included fixture files (as expected by `check_fixtures_test.go`):

- `log4netdotnet20` — log4net .NET 2.0 build (named after the original DLL inside)
- `log4netdotnet462` — log4net .NET 4.6.2 build
- `puttywin32x86` — PuTTY 32-bit (x86) executable(s)
- `puttywin64x64` — PuTTY 64-bit (x64) executable(s)
- `puttywin64arm` — PuTTY Windows ARM64 executable(s)
- `sqlite3win32x86` — SQLite 32-bit (x86) DLL/package
- `sqlite3win64x64` — SQLite 64-bit (x64) DLL/package
- `sqlite3win64arm` — SQLite Windows ARM64 DLL/package

Where the files come from

- The `log4net` assemblies were obtained from the NuGet package: [log4net on NuGet](https://www.nuget.org/packages/log4net/)
- Putty builds: [the.earth.li putty latest mirror](https://the.earth.li/~sgtatham/putty/latest/) (and PuTTY project page)
- SQLite prebuilt Windows DLL ZIPs: [sqlite.org downloads](https://www.sqlite.org/download.html)

Licenses

- log4net: Apache License 2.0. See the package page: [log4net on NuGet](https://www.nuget.org/packages/log4net/) and the license text: [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)
- PuTTY: MIT/X11-style license. See [PuTTY licence page](https://www.chiark.greenend.org.uk/~sgtatham/putty/licence.html)
- SQLite: public-domain (or permissive where necessary). See [SQLite copyright page](https://www.sqlite.org/copyright.html)
