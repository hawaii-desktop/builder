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

package steps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"string"
	"time"
)

type SrpmBuild struct {
	BuildStep
	TopDir    string
	BuildDir  string
	RpmDir    string
	SourceDir string
	SpecDir   string
	SrcrpmDir string
	SpecFile  string
}

func (s *SrpmBuild) Run() *BuildResult {
	// Build in the current workind directory if not otherwise specified
	var (
		topdir
		builddir
		rpmdir
		sourcedir
		specdir
		srcrpmdir
	)
	if topdir = s.TopDir; topdir == "" {
		topdir = s.Cwd
	}
	if builddir = s.BuildDir; builddir == "" {
		builddir = s.Cwd
	}
	if rpmdir = s.BuildDir; rpmdir == "" {
		rpmdir = s.Cwd
	}
	if sourcedir = s.SourceDir; sourcedir == "" {
		sourcedir = s.Cwd
	}
	if specdir = s.SpecDir; specdir == "" {
		specdir = s.Cwd
	}
	if srcrpmdir = s.SrcrpmDir; srcrpmdir == "" {
		srcrpmdir = s.Cwd
	}

	// Prepare arguments
	var args []string
	args = append(args, fmt.Sprintf("--define \"_topdir %s\"", topdir))
	args = append(args, fmt.Sprintf("--define \"_builddir %s\"", builddir))
	args = append(args, fmt.Sprintf("--define \"_rpmdir %s\"", rpmdir))
	args = append(args, fmt.Sprintf("--define \"_sourcedir %s\"", sourcedir))
	args = append(args, fmt.Sprintf("--define \"_specdir %s\"", specdir))
	args = append(args, fmt.Sprintf("--define \"_srcrpmdir %s\"", srcrpmdir))

	// Append git information
	if s.Parent.Properties.GetBool("VcsBuild", false) {
		date := s.Parent.Properties.GetString("GitDate")
		revision := s.Parent.Properties.GetString("GitShortRev")
		args = append(args, fmt.Sprintf("--define \"_checkout %sgit%s\"", date, revision))
	}

	// Append specfile
	args = append(args, "-bs", s.SpecFile)

	return s.ExecShellCommand("rpmbuild", args, []string{})
}

func (s *SrpmBuild) Finalize() {
	// Lines with these prefixes will be marked as errors
	var prefixes = []string{"   ", "RPM build errors:", "error: "}

	// Errors list
	errors := make([]string, 0)

	// Scan all output lines
	for {
		// Read line, exit the loop when there's nothing left to read
		line, err := stdout.ReadString('\n')
		if err != nil {
			break
		}

		// Check for errors
		for prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				errors = append(errors, line)
			}
		}

		// Save SRPM name
		re, _ := regexp.Compile("^Wrote: .*/([^/]*.src.rpm)")
		m := re.FindStringSubmatch(line)
		if len(m) > 0 {
			s.Parent.Properties["SRPM"] = m[1]
		}
	}

	// Add a summary for errors
	if len(errors) > 0 {
		s.AddSummary("SRPM Errors", strings.Join(errors, "\n"))
	}
}
