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

package slave

import (
	"fmt"
	"github.com/hawaii-desktop/builder/utils"
	"os"
	"os/exec"
	"path"
	"time"
)

func NewImageFactory(j *Job) *Factory {
	f := NewFactory(j)

	// Fetch the repository
	f.AddBuildStep(&BuildStep{
		Name:      fmt.Sprintf("git %s", j.Info.Image.VcsUrl),
		KeepGoing: false,
		Run:       imgFactoryGitFetch,
	})

	// Flatten kickstart
	f.AddBuildStep(&BuildStep{
		Name:      "ksflatten",
		KeepGoing: false,
		Run:       imgFactoryFlatten,
	})

	// Flatten kickstart
	f.AddBuildStep(&BuildStep{
		Name:      "build",
		KeepGoing: false,
		Run:       imgFactoryBuild,
	})

	return f
}

func imgFactoryGitFetch(bs *BuildStep) error {
	// Clone or update
	url := bs.parent.job.Info.Image.VcsUrl
	branch := bs.parent.job.Info.Image.VcsBranch
	err := bs.parent.DownloadGit(url, branch, bs.parent.workdir, "sources")
	if err != nil {
		return err
	}

	return nil
}

func imgFactoryFlatten(bs *BuildStep) error {
	// Need to run from the sources
	os.Chdir(path.Join(bs.parent.workdir, "sources"))

	// Determine the source kickstart
	filename := "hawaii-livecd.ks"
	if bs.parent.job.Architecture == "armhfp" {
		filename = "hawaii-arm.ks"
	}

	// Flatten
	cmd := bs.parent.Command("ksflatten", "-c", filename, "-o", "../flattened.ks")
	if err := utils.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}
	_, err := os.Stat("../flattened.ks")
	if err != nil {
		return err
	}

	return nil
}

func imgFactoryBuild(bs *BuildStep) error {
	today := time.Now().Format("%Y%m%d")
	fsname := fmt.Sprintf("hawaii-%s-%s", today, bs.parent.job.Architecture)
	filename := fsname

	// Build
	var cmd *exec.Cmd
	if bs.parent.job.Architecture == "armhfp" {
		cmd = bs.parent.Command("sudo", "appliance-creator",
			"--logfile", "results/appliance.log", "--cache", "cache",
			"-d", "-v", "-o", "results", "--format=raw", "--checksum",
			"--name", filename, "--version", "22", "--release", today,
			"-c", "flattened.ks")
		filename += ".raw"
	} else {
		cmd = bs.parent.Command("sudo", "livecd-creator", "--releasever=22",
			"--title=Hawaii", "--product=Hawaii", "-c", "flattened.ks",
			"-f", fsname, "-d", "-v", "--cache", "cache", "--tmpdir", "tmp")
		filename += ".iso"
	}
	if err := utils.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}
	_, err := os.Stat(filename)
	if err != nil {
		return err
	}

	return nil
}
