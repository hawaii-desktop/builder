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

package master

import (
	"fmt"
	"github.com/hawaii-desktop/builder/logging"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// Create or update repodata and repoview from the rootrepodir directory.
func (m *Master) updateRepoData(rootrepodir string) {
	// Iterate over the releases
	releaserootdir := filepath.Join(rootrepodir, "fedora", "releases")
	releases, _ := ioutil.ReadDir(releaserootdir)
	for _, release := range releases {
		if release.IsDir() {
			// Iterate over the architectures for this release
			archrootdir := filepath.Join(releaserootdir, release.Name(), "Everything")
			archs, _ := ioutil.ReadDir(archrootdir)
			for _, arch := range archs {
				if arch.IsDir() && arch.Name() != "source" {
					osdir := filepath.Join(archrootdir, arch.Name(), "os")
					var cmd *exec.Cmd

					// Create repository data
					cachedir := filepath.Join(osdir, ".cache")
					cmd = exec.Command("createrepo", "--verbose", "--compress-type", "xz",
						"--update", "--deltas", "--num-deltas", "5",
						"--cachedir", cachedir, osdir)
					if output, err := cmd.CombinedOutput(); err != nil {
						logging.Errorf("Failed to create repodata for %s: %s\n%s", osdir, err, string(output))
					}

					// Remove old repository view
					os.RemoveAll(filepath.Join(osdir, "repoview"))

					// Create repository view
					cmd = exec.Command("repoview", "-t", fmt.Sprintf("Hawaii for %s-%s", release.Name(), arch.Name()), osdir)
					if output, err := cmd.CombinedOutput(); err != nil {
						logging.Errorf("Failed to create repoview for %s: %s\n%s", osdir, err, string(output))
					}
				}
			}
		}
	}
}

// Update repodata for the main repository every time another
// goroutine ask to do it. Eventually return when false is queued
// to the channel.
func (m *Master) processRepoDataUpdates(repoc <-chan bool) {
	for {
		select {
		case result := <-repoc:
			if result {
				m.updateRepoData(Config.Storage.MainRepoDir)
			} else {
				return
			}
		}
	}
}
