# testdata: PE-files for test fixtures

This directory contains binary test files for fixtures used by the tests in this repository. These valid PE-files are only used in test fixtures. This repository does not redistribute them.

## Included fixture files (as expected by `check_fixtures_test.go`)

- `log4netdotnet20` — log4net .NET 2.0 build (CLR DLL)
- `log4netdotnet462` — log4net .NET 4.6.2 build (CLR DLL)
- `NSISPortable311` — NSIS Portable Version 3.11 executable
- `puttywin32x86` — PuTTY 32-bit (x86) executable
- `puttywin64x64` — PuTTY 64-bit (x64) executable
- `puttywin64arm` — PuTTY Windows ARM64 executable
- `sqlite3win32x86` — SQLite 32-bit (x86) DLL
- `sqlite3win64x64` — SQLite 64-bit (x64) DLL
- `sqlite3win64arm` — SQLite Windows ARM64 DLL
- `somepng` — a PNG image file

## Where the files come from

- log4net assemblies: [log4net on NuGet](https://www.nuget.org/packages/log4net/)
- Putty executables: [the.earth.li putty latest mirror](https://the.earth.li/~sgtatham/putty/latest/) (and PuTTY project page)
- SQLite prebuilt Windows DLLs: [sqlite.org downloads](https://www.sqlite.org/download.html)
- NSIS Portable: [PortableApps.com](https://portableapps.com/apps/development/nsis_portable) resp. [NSIS Project page](https://nsis.sourceforge.io/Main_Page)

## Licenses

- log4net: Apache License 2.0 ("[...] Subject to the terms and conditions of this License, each Contributor hereby grants to You a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable copyright license to reproduce, prepare Derivative Works of, publicly display, publicly perform, sublicense, and distribute the Work and such Derivative Works in Source or Object form. [...]"). See the package page: [log4net on NuGet](https://www.nuget.org/packages/log4net/) and the license text: [Apache License 2.0](https://licenses.nuget.org/Apache-2.0)
- PuTTY: MIT-style license ("PuTTY is copyright 1997-2025 Simon Tatham. Portions copyright Robert de Bath, Joris van Rantwijk, Delian Delchev, Andreas Schultz, Jeroen Massar, Wez Furlong, Nicolas Barry, Justin Bradford, Ben Harris, Malcolm Smith, Ahmad Khalifa, Markus Kuhn, Colin Watson, Christopher Staite, Lorenz Diener, Christian Brabandt, Jeff Smith, Pavel Kryukov, Maxim Kuznetsov, Svyatoslav Kuzmich, Nico Williams, Viktor Dukhovni, Josh Dersch, Lars Brinkhoff, and CORE SDI S.A. Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions: The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software. [...]"). See [PuTTY licence page](https://www.chiark.greenend.org.uk/~sgtatham/putty/licence.html)
- SQLite: public-domain ("[...] Anyone is free to copy, modify, publish, use, compile, sell, or distribute the original SQLite code, either in source code form or as a compiled binary, for any purpose, commercial or non-commercial, and by any means. [...]"). See [SQLite copyright page](https://www.sqlite.org/copyright.html)
- NSISPortable311: Source and most components provided under the zlib/libpng license ("This software is provided 'as-is'... Permission is granted to anyone to use this software for any purpose, including commercial applications, and to alter it and redistribute it freely"). Some compression modules bundled with NSIS are licensed under other terms (bzip2, LZMA/Common Public License). See [NSIS license page](https://nsis.sourceforge.io/License)
