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
	"bytes"
	"fmt"
	"github.com/hawaii-desktop/builder/logging"
	"io/ioutil"
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
	cmd := exec.Command("ksflatten", "-c", filename, "-o", "../flattened.ks")
	if err := bs.parent.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}
	_, err := os.Stat("../flattened.ks")
	if err != nil {
		return err
	}

	return nil
}

func imgFactoryBuild(bs *BuildStep) error {
	// Get the context data
	d, ok := FromContext(bs.parent.job.ctx)
	if !ok {
		logging.Fatalln("Internal error: no data from context")
	}

	today := time.Now().Format("%Y%m%d")
	fsname := fmt.Sprintf("hawaii-%s-%s", today, bs.parent.job.Architecture)
	filename := fsname

	// Fedora release
	releasever := "22"

	// Replace @REPO_URL@
	input, err := ioutil.ReadFile("flattened.ks")
	if err != nil {
		return err
	}
	lines := bytes.Split(input, []byte("\n"))
	for i, line := range lines {
		if bytes.Contains(line, []byte("@REPO_URL@")) {
			lines[i] = bytes.Replace(line, []byte("@REPO_URL@"), []byte(d.RepoUrl), -1)
		}
	}
	output := bytes.Join(lines, []byte("\n"))
	if err = ioutil.WriteFile("flattened.ks", output, 0644); err != nil {
		return err
	}

	// Build
	var cmd *exec.Cmd
	if bs.parent.job.Architecture == "armhfp" {
		cmd = exec.Command("sudo", "appliance-creator",
			"--logfile", "results/appliance.log", "--cache", "cache",
			"-d", "-v", "-o", "results", "--format=raw", "--checksum",
			"--name", filename, "--version", releasever, "--release", today,
			"-c", "flattened.ks")
		filename += ".raw"
	} else {
		linuxcmd := "linux64"
		if bs.parent.job.Architecture == "i386" {
			linuxcmd = "linux32"
		}

		cmd = exec.Command("sudo", linuxcmd, "livecd-creator", "--releasever="+releasever,
			"--title=Hawaii", "--product=Hawaii", "-c", "flattened.ks",
			"-f", fsname, "-d", "-v", "--cache", "cache", "--tmpdir", "tmp")
		filename += ".iso"
	}
	if err := bs.parent.RunCommand(cmd); err != nil {
		return err
	}
	_, err = os.Stat(filename)
	if err != nil {
		return err
	}

	return nil
}
