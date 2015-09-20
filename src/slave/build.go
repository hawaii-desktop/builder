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
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/hawaii-desktop/builder/src/utils"
	"io/ioutil"
	"log"
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
	// Logger for capturing command output.
	logger *log.Logger
	// Output.
	buffer *bytes.Buffer
	// Log streamer for stdout.
	logStreamerOut *logging.LogStreamer
	// Log streamer for stderr.
	logStreamerErr *logging.LogStreamer
}

// Build step running function.
type BuildStepRunFunc func(bs *BuildStep) error

// Single build step that is part of a factory.
// This defines the properties shared by all build steps.
type BuildStep struct {
	// Factory that owns this build step.
	parent *Factory
	// Summary with the most important log messages.
	summary map[string]string
	// Name of this build step.
	Name string
	// Indicates whether a failure should keep the factory going.
	KeepGoing bool
	// Running function.
	Run BuildStepRunFunc
}

// Create a new factory.
func NewFactory(j *Job) *Factory {
	// Create working directory
	workdir := path.Join(Config.Directory.WorkDir, j.Target.Name)
	os.MkdirAll(workdir, 0755)

	// Logging
	var buffer bytes.Buffer
	logger := log.New(&buffer, "", log.Ldate|log.Ltime)
	logStreamerOut := logging.NewLogStreamer(logger, "STDOUT ")
	logStreamerErr := logging.NewLogStreamer(logger, "STDERR ")

	// Factory
	return &Factory{
		job:            j,
		workdir:        workdir,
		steps:          make([]*BuildStep, 0),
		properties:     make(VariantMap),
		logger:         logger,
		buffer:         &buffer,
		logStreamerOut: logStreamerOut,
		logStreamerErr: logStreamerErr,
	}
}

// Create a shell command and capture the output.
// Return a Command pointer ready to be executed.
func (f *Factory) Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)

	cmd.Stdout = f.logStreamerOut
	cmd.Stderr = f.logStreamerErr

	cmd.Stdout.Write([]byte(strings.Repeat("*", 79) + "\n"))
	cmd.Stdout.Write([]byte(fmt.Sprintf("Running: %s\n", strings.Join(cmd.Args, " "))))
	cmd.Stdout.Write([]byte(fmt.Sprintf("Argv: %q\n", cmd.Args)))
	cwd, _ := os.Getwd()
	cmd.Stdout.Write([]byte(fmt.Sprintf("From: %s\n", cwd)))
	cmd.Stdout.Write([]byte(strings.Repeat("*", 79) + "\n"))

	return cmd
}

// Create a shell command with a custom environment and capture the output.
// Return a Command pointer ready to be executed.
func (f *Factory) CommandWithEnv(env []string, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env = env

	cmd.Stdout = f.logStreamerOut
	cmd.Stderr = f.logStreamerErr

	cmd.Stdout.Write([]byte(strings.Repeat("*", 79) + "\n"))
	cmd.Stdout.Write([]byte("Environment:\n"))
	for _, e := range cmd.Env {
		cmd.Stdout.Write([]byte(fmt.Sprintf("\t%s\n", e)))
	}
	cmd.Stdout.Write([]byte(fmt.Sprintf("Running: %s\n", strings.Join(cmd.Args, " "))))
	cmd.Stdout.Write([]byte(fmt.Sprintf("Argv: %q\n", cmd.Args)))
	cwd, _ := os.Getwd()
	cmd.Stdout.Write([]byte(fmt.Sprintf("From: %s\n", cwd)))
	cmd.Stdout.Write([]byte(strings.Repeat("*", 79) + "\n"))

	return cmd
}

// Clone or pull a git repository.
func (f *Factory) DownloadGit(url, tag, parentdir, clonedirname string) error {
	// Clone if the clone directory doesn't exist otherwise pull
	if _, err := ioutil.ReadFile(path.Join(parentdir, clonedirname, ".git", "HEAD")); err != nil {
		// Clone repository
		os.Chdir(parentdir)
		cmd := f.Command("git", "clone", url, clonedirname)
		if err := utils.RunWithTimeout(cmd, cloneTimeout); err != nil {
			return err
		}
	}

	// Enter the clone directory
	os.Chdir(path.Join(parentdir, clonedirname))

	// Fetch from origin
	cmd := f.Command("git", "fetch", "origin")
	if err := utils.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	// Checkout tag or branch
	cmd = f.Command("git", "checkout", tag)
	if err := utils.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	// Pull from an existing clone
	cmd = f.Command("git", "pull")
	if err := utils.RunWithTimeout(cmd, cloneTimeout); err != nil {
		return err
	}

	return nil
}

// Close the factory.
func (f *Factory) Close() {
	f.logStreamerOut.Close()
	f.logStreamerErr.Close()
}

// Append the build step.
// Automatically set parent and initialize the summary.
func (f *Factory) AddBuildStep(bs *BuildStep) {
	f.sMutex.Lock()
	defer f.sMutex.Unlock()
	bs.parent = f
	bs.summary = make(map[string]string)
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

		// Run the step
		logging.Infof("=> Running build step \"%s\"\n", bs.Name)
		err := bs.Run(bs)

		// Send the build step log to the master and reset
		logging.Infoln(f.buffer.String())
		f.buffer.Reset()

		// Flush logs
		f.logStreamerOut.FlushRecord()
		f.logStreamerErr.FlushRecord()

		// Check the result
		if err != nil && !bs.KeepGoing {
			logging.Errorf("Build step \"%s\" failed: %s\n", bs.Name, err)
			return false
		}

		// Elapsed time
		elapsed := time.Since(start)
		logging.Traceln(elapsed)
	}

	return true
}

// Add a summary entry that will be shown by the user interface.
func (bs *BuildStep) AddSummary(label string, value string) {
	bs.summary[label] = value
}
