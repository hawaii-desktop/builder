/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
 *
 * Author(s):
 *    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * $BEGIN_LICENSE:AGPL3+$
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * $END_LICENSE$
 ***************************************************************************/

package version

import (
	"fmt"
	"io"
	"os"
)

// Output the version string to the writer, in the following
// format, followed by a new line:
//
//     <cmd> <package> <version>
//
// Where <cmd> is the command, <package> the canonical project
// name and <version> the actual version string.
//
// For example, a binary "builder-master" built from
// github.com/hawaii-desktop/builder with version v2.0 would
// output the following:
//
//     binary-master github.com/hawaii-desktop/builder v2.0
//
func FprintVersion(w io.Writer) {
	fmt.Fprintln(w, os.Args[0], Package, Version)
}

// Output version information using FprintVersion() on
// standard output.
func PrintVersion() {
	FprintVersion(os.Stdout)
}
