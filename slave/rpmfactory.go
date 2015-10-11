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
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

var (
	ErrInvalidRepository = errors.New("invalid repository URL")
	ErrNoVcsInformation  = errors.New("vcs information was not saved")
	ErrNoSrpm            = errors.New("Srpm property was not saved")
)

func NewRpmFactory(j *Job) *Factory {
	f := NewFactory(j)

	// Make the repositories iterable
	var repos [][]string
	repos = append(repos, []string{"packaging", j.Info.Package.VcsUrl, j.Info.Package.VcsBranch})
	if j.Info.Package.Ci {
		repos = append(repos, []string{j.Target, j.Info.Package.UpstreamVcsUrl, j.Info.Package.UpstreamVcsBranch})
	}

	// Fetch all repositories
	for i := range repos {
		repo := repos[i]
		f.AddBuildStep(&BuildStep{
			Name:      fmt.Sprintf("git %s", repo[0]),
			KeepGoing: false,
			Run: func(bs *BuildStep) error {
				return rpmFactoryGitFetch(repo, bs)
			},
		})
	}

	// TODO: Do we need to build?
	buildNeeded := true
	if !buildNeeded {
		return f
	}

	// Validate spec file with rpmlint but do not block builds on failure
	f.AddBuildStep(&BuildStep{
		Name:      "rpmlint",
		KeepGoing: true,
		Run:       rpmFactoryRpmlint,
	})

	// Create a tar.xz for continuous integration packages
	// or run spectool to download the sources
	if j.Info.Package.Ci {
		f.AddBuildStep(&BuildStep{
			Name:      "source tarball",
			KeepGoing: false,
			Run:       rpmFactorySources,
		})
	} else {
		f.AddBuildStep(&BuildStep{
			Name:      "source tarball",
			KeepGoing: false,
			Run:       rpmFactorySpectool,
		})
	}

	// Build SRPM
	f.AddBuildStep(&BuildStep{
		Name:      "build srpm",
		KeepGoing: false,
		Run:       rpmFactorySrpmBuild,
	})

	// Rebuild SRPM
	f.AddBuildStep(&BuildStep{
		Name:      "mock rebuild",
		KeepGoing: false,
		Run:       rpmFactoryMockRebuild,
	})

	return f
}

func rpmFactoryGitFetch(repo []string, bs *BuildStep) error {
	// Clone or update
	err := bs.parent.DownloadGit(repo[1], repo[2], bs.parent.workdir, repo[0])
	if err != nil {
		return err
	}

	// Get version information from upstream
	if repo[0] == bs.parent.job.Target {
		cmd := exec.Command("git", "log", "-1", `--format="%cd"`, "--date=short")
		output, err := bs.parent.RunCombinedWithTimeout(cmd, cloneTimeout)
		if err != nil {
			return err
		}
		result := strings.Replace(string(output), "-", "", -1)
		result = strings.Replace(result, "\"", "", -1)
		bs.parent.properties["VcsDate"] = strings.TrimSuffix(result, "\n")

		cmd = exec.Command("git", "log", "-1", `--format="%h"`)
		output, err = bs.parent.RunCombinedWithTimeout(cmd, cloneTimeout)
		if err != nil {
			return err
		}
		result = strings.Replace(string(output), "\"", "", -1)
		bs.parent.properties["VcsShortRev"] = strings.TrimSuffix(result, "\n")
	}

	return nil
}

func rpmFactoryRpmlint(bs *BuildStep) error {
	// Change directory
	cwd := path.Join(bs.parent.workdir, "packaging")
	os.Chdir(cwd)

	// Run rpmlint
	cmd := exec.Command("rpmlint", "-i", bs.parent.job.Target+".spec")
	output, err := bs.parent.RunCombinedWithTimeout(cmd, cloneTimeout)
	if err != nil {
		return err
	}

	// Count warnings and errors
	var (
		errors   []string
		warnings []string
	)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// Remove logging prefix
		line := scanner.Text()

		// Check prefix
		if strings.HasPrefix(line, "E: ") {
			errors = append(errors, line)
		} else if strings.HasPrefix(line, "W: ") {
			warnings = append(warnings, line)
		}
	}

	// Add a summary
	if len(errors) > 0 {
		bs.AddSummary("Rpmlint Errors", strings.Join(errors, "\n"))
	}
	if len(warnings) > 0 {
		bs.AddSummary("Rpmlint Warnings", strings.Join(warnings, "\n"))
	}

	return nil
}

func rpmFactorySources(bs *BuildStep) error {
	// Make sources
	filename := path.Join("packaging", bs.parent.job.Target+".tar.xz")
	cmd := exec.Command("tar", "-cJf", filename, bs.parent.job.Target)
	if err := bs.parent.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}
	_, err := os.Stat(filename)
	if err != nil {
		return err
	}

	return nil
}

func rpmFactorySpectool(bs *BuildStep) error {
	// Change directory
	cwd := path.Join(bs.parent.workdir, "packaging")
	os.Chdir(cwd)

	// Make sources
	filename := bs.parent.job.Target + ".spec"
	cmd := exec.Command("spectool", "-g", "-A", filename)
	if err := bs.parent.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}
	_, err := os.Stat(filename)
	if err != nil {
		return err
	}

	return nil
}

func rpmFactorySrpmBuild(bs *BuildStep) error {
	// Change directory
	cwd := path.Join(bs.parent.workdir, "packaging")
	os.Chdir(cwd)

	// Prepare arguments
	var args []string
	args = append(args, "--define", "_sourcedir "+cwd)
	args = append(args, "--define", "_specdir "+cwd)
	args = append(args, "--define", "_builddir "+cwd)
	args = append(args, "--define", "_srcrpmdir "+cwd)
	args = append(args, "--define", "_rpmdir "+cwd)
	args = append(args, "--define", "_buildrootdir "+cwd)
	args = append(args, "--define", "_topdir "+cwd)

	// Append git information
	if bs.parent.job.Info.Package.Ci {
		date := bs.parent.properties.GetString("VcsDate", "")
		revision := bs.parent.properties.GetString("VcsShortRev", "")
		if date == "" || revision == "" {
			return ErrNoVcsInformation
		}
		args = append(args, "--define", "_checkout "+fmt.Sprintf("%sgit%s", date, revision))
	}

	// Append specfile
	args = append(args, "-bs", bs.parent.job.Target+".spec")

	// Run rpmbuild
	cmd := exec.Command("rpmbuild", args...)
	output, err := bs.parent.RunCombinedWithTimeout(cmd, cloneTimeout)
	if err != nil {
		return err
	}

	// Lines with these prefixes will be marked as errors
	var prefixes = []string{"   ", "RPM build errors:", "error: "}

	// Errors list
	var errors []string

	// Scan all output lines
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// Remove logging prefix
		line := scanner.Text()

		// Check for errors
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				errors = append(errors, line)
			}
		}

		// Save SRPM name
		re, _ := regexp.Compile("^Wrote: .*/([^/]*.src.rpm)")
		m := re.FindStringSubmatch(line)
		if len(m) > 0 {
			bs.parent.properties["Srpm"] = m[1]
		}
	}

	// Add a summary for errors
	if len(errors) > 0 {
		bs.AddSummary("SRPM Errors", strings.Join(errors, "\n"))
	}

	return nil
}

func rpmFactoryMockRebuild(bs *BuildStep) error {
	// Change directory
	cwd := path.Join(bs.parent.workdir, "packaging")
	os.Chdir(cwd)

	root := fmt.Sprintf("fedora-%s-%s", "22", "x86_64")

	args := []string{"--root", root, "-m", "--resultdir=../results"}
	if bs.parent.job.Info.Package.Ci {
		date := bs.parent.properties.GetString("VcsDate", "")
		revision := bs.parent.properties.GetString("VcsShortRev", "")
		if date == "" || revision == "" {
			return ErrNoVcsInformation
		}
		rev := date + "git" + revision
		args = append(args, "-m", "--define=_checkout "+rev)
	}
	args = append(args, "-m", `--define="vendor Hawaii"`)
	args = append(args, "-m", `--define="packager Hawaii"`)
	args = append(args, "-m", `--define="distribution Hawaii"`)
	srpm := bs.parent.properties.GetString("Srpm", "")
	if srpm == "" {
		return ErrNoSrpm
	}
	args = append(args, srpm)
	args = append(args, "--tmp_prefix", "builder-mock")

	// We run mockchain instead of mock so we can use the remote
	// repository from master, we also need to have /sbin come first
	// so the right mock is executed (the one that doesn't ask for a
	// password, provided that the user is in the mock group)
	cmd := exec.Command("mockchain", args...)
	cmd.Env = []string{"PATH=/usr/local/sbin:/sbin:/usr/sbin:/usr/local/bin:/bin:/usr/bin"}
	if err := bs.parent.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	return nil
}
