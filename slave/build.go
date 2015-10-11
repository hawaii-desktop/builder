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
	"regexp"
	"strings"
	"sync"
	"time"
)

// VCS
var (
	lsRemoteTimeout = 5 * time.Minute
	cloneTimeout    = 10 * time.Minute
	fetchTimeout    = 5 * time.Minute
	checkoutTimeout = 1 * time.Minute
	UrlRegExp       = regexp.MustCompile(`^(?P<scheme>\w+://)*(?P<user>.+@)*(?P<host>[\w\d\.]+)(?P<port>:[\d]+){0,1}/*(?P<path>(?P<dir>[\w.]+)/*(?P<repo>[\w.]+))$`)
)

// List of build steps to be executed one after another.
type Factory struct {
	// Pointer to the job that can be update while running.
	job *Job
	// Working directory for this job.
	workdir string
	// Build steps of this factory.
	steps []*BuildStep
	// Properties.
	properties VariantMap
	// Protects the build steps array.
	sMutex sync.Mutex
	// Output.
	buffer *bytes.Buffer
}

// Build step running function.
type BuildStepRunFunc func(bs *BuildStep) error

// Single build step that is part of a factory.
// This defines the properties shared by all build steps.
type BuildStep struct {
	// Name of this build step.
	Name string
	// Indicates whether a failure should keep the factory going.
	KeepGoing bool
	// Running function.
	Run BuildStepRunFunc
	// Factory that owns this build step.
	parent *Factory
	// Summary with the most important log messages.
	summary map[string][]string
	// When the step has been started.
	started time.Time
	// When the step has finished.
	finished time.Time
	// Output capture from execution.
	output []byte
	// Additional logs.
	logs map[string][]byte
}

// Create a new factory.
func NewFactory(j *Job) *Factory {
	// Create working directory
	workdir := path.Join(Config.Directory.WorkDir, j.Target)
	os.MkdirAll(workdir, 0755)

	// Factory
	var buffer bytes.Buffer
	return &Factory{
		job:        j,
		workdir:    workdir,
		steps:      make([]*BuildStep, 0),
		properties: make(VariantMap),
		buffer:     &buffer,
	}
}

// Run a command with timeout.
func (f *Factory) RunWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	cwd, _ := os.Getwd()

	if len(cmd.Env) > 0 {
		fmt.Fprintf(f.buffer, "Environment:\n")
		for _, e := range cmd.Env {
			fmt.Fprintf(f.buffer, "\t%s\n", e)
		}
	}
	fmt.Fprintf(f.buffer, "Running: %s\n", strings.Join(cmd.Args, " "))
	fmt.Fprintf(f.buffer, "Argv: %q\n", cmd.Args)
	fmt.Fprintf(f.buffer, "From: %s\n", cwd)

	t := time.AfterFunc(timeout, func() { cmd.Process.Kill() })
	defer t.Stop()

	output, err := cmd.CombinedOutput()
	fmt.Fprintf(f.buffer, string(output))

	return err
}

func (f *Factory) RunCombinedWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	cwd, _ := os.Getwd()

	if len(cmd.Env) > 0 {
		fmt.Fprintf(f.buffer, "Environment:\n")
		for _, e := range cmd.Env {
			fmt.Fprintf(f.buffer, "\t%s\n", e)
		}
	}
	fmt.Fprintf(f.buffer, "Running: %s\n", strings.Join(cmd.Args, " "))
	fmt.Fprintf(f.buffer, "Argv: %q\n", cmd.Args)
	fmt.Fprintf(f.buffer, "From: %s\n", cwd)

	t := time.AfterFunc(timeout, func() { cmd.Process.Kill() })
	defer t.Stop()

	output, err := cmd.CombinedOutput()
	fmt.Fprintf(f.buffer, string(output))

	return output, err
}

// Clone or pull a git repository.
func (f *Factory) DownloadGit(url, tag, parentdir, clonedirname string) error {
	// Clone if the clone directory doesn't exist otherwise pull
	if _, err := ioutil.ReadFile(path.Join(parentdir, clonedirname, ".git", "HEAD")); err != nil {
		// Clone repository
		os.Chdir(parentdir)
		cmd := exec.Command("git", "clone", url, clonedirname)
		if err := f.RunWithTimeout(cmd, cloneTimeout); err != nil {
			return err
		}
	}

	// Enter the clone directory
	os.Chdir(path.Join(parentdir, clonedirname))

	// Fetch from origin
	cmd := exec.Command("git", "fetch", "origin")
	if err := f.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	// Checkout tag or branch
	cmd = exec.Command("git", "checkout", tag)
	if err := f.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	// Pull from an existing clone
	cmd = exec.Command("git", "pull")
	if err := f.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	return nil
}

// Close the factory.
func (f *Factory) Close() {
}

// Append the build step.
// Automatically set parent and initialize the summary.
func (f *Factory) AddBuildStep(bs *BuildStep) {
	f.sMutex.Lock()
	defer f.sMutex.Unlock()
	bs.parent = f
	bs.summary = make(map[string][]string)
	bs.logs = make(map[string][]byte)
	f.steps = append(f.steps, bs)
}

// Run the build steps in the same order in which they were added.
// Returns true if all of them were successful, otherwise false.
// Stops when a build step fails unless it has the KeepGoing flag.
func (f *Factory) Run() bool {
	f.sMutex.Lock()
	defer f.sMutex.Unlock()

	for _, bs := range f.steps {
		// Start measuring time
		start := time.Now()

		// Change directory
		os.Chdir(f.workdir)

		// Send the update and run the step
		logging.Infof("=> Running build step \"%s\"\n", bs.Name)
		bs.started = start
		f.job.stepUpdateQueue <- bs
		err := bs.Run(bs)

		// Elapsed time
		elapsed := time.Since(start)
		bs.finished = start.Add(elapsed)

		// Collect output and send the update
		bs.output = f.buffer.Bytes()
		f.buffer.Reset()
		f.job.stepUpdateQueue <- bs

		// Check the result
		if err == nil {
			logging.Infof("<= Build step \"%s\" took %v\n", bs.Name, elapsed)
		} else {
			logging.Errorf("<= Build step \"%s\" failed in %v: %s\n", bs.Name, elapsed, err)
			if !bs.KeepGoing {
				return false
			}
		}
	}

	return true
}

// Add a summary entry that will be shown by the user interface.
func (bs *BuildStep) AddSummary(label string, value string) {
	bs.summary[label] = append(bs.summary[label], value)
}
