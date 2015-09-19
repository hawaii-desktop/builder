/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015 Pier Luigi Fiorini
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

package vcs

import (
	"github.com/hawaii-desktop/builder/src/utils"
	"io/ioutil"
	"path"
)

func DownloadGit(url string, tag string, dir string) (string, error) {
	if _, err := ioutil.ReadFile(path.Join(dir, ".git/HEAD")); err != nil {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return "", err
		}

		var args = []string{"clone", url, dir}
		if output, err := utils.ExecShellCommandWithTimeout("git", args, []string, cloneTimeout); err != nil {
			return output, err
		}
	}
}
