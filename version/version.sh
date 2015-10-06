#!/bin/sh

# This script outputs the current content of version.go, using git describe.
# Redirect the output to the target file.
# Running this script is generally needed only to update after releases.
# The actual value will be replaced at build time if make is issued.

set -e

cat <<EOF
/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015 Pier Luigi Fiorini
 *
 * Author(s):
 *    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * \$BEGIN_LICENSE:AGPL3+$
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
 * \$END_LICENSE$
 ***************************************************************************/

package version

// Canonical project import path under which the package was built.
var Package = "$(go list)"

// Version of the binary. This is set to the latest release git tag, always
// suffixed by "+unknown".  It will be replaced by the actual version at
// build time.  The following value will be used if the programm is run
// after a go get based installation.
var Version = "$(git describe --match 'v[0-9]*' --dirty='.m' --always)+unknown"
EOF
