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
	"os"
	"os/exec"
	"sync"
	"time"
)

// List of build steps to be executed one after another.
type Factory struct {
	// Build steps of this factory.
	steps []*BuildStep
	// Properties.
	properties VariantMap
	// Protects the build steps array.
	sMutex sync.Mutex
}

// Create a new factory.
func NewFactory() *Factory {
	f := &Factory{
		steps:      make([]*BuildStep, 0),
		properties: make(VariantMap),
	}
	return f
}

// Create a new build step.
func (f *Factory) NewBuildStep(name string, cwd string, keepGoing bool) *BuildStep {
	b := &BuildStep{
		parent:    nil,
		name:      name,
		cwd:       cwd,
		summary:   make(map[string]string),
		keepGoing: keepGoing,
	}
	return b
}

// Append the build step.
func (f *Factory) AddBuildStep(b *BuildStep) {
	f.sMutex.Lock()
	defer f.sMutex.Unlock()
	b.parent = f
	f.steps = append(f.steps, b)
}

// Run the build steps in the same order in which they were added.
// Returns true if all of them were successful, otherwise false.
// Stops when a build step fails unless it has the KeepGoing flag.
func (f *Factory) Run() bool {
	f.sMutex.Lock()
	defer f.sMutex.Unlock()

	var result *BuildResult = nil
	for _, b := range f.steps {
		result = b.Run()
		if !result.IsSuccessful() && !b.ShouldKeepGoing() {
			return false
		}
		b.Finalize()
	}

	return true
}

// Result of a build step.
type BuildResult struct {
	// Standard output.
	stdout string
	// Standard error.
	stderr string
	// Indicates whether the step was successful or not.
	success bool
	// How much time did the step take.
	elapsedTime time.Duration
}

// Return standard output.
func (r *BuildResult) Stdout() string {
	return r.stdout
}

// Return standard error.
func (r *BuildResult) Stderr() string {
	return r.stderr
}

// Return whether the step was successful or not.
func (r *BuildResult) IsSuccessful() bool {
	return r.success
}

// Return how much time the step took.
func (r *BuildResult) ElapsedTime() time.Duration {
	return r.elapsedTime
}

// Interface for a single build step that is part of a factory.
// This defines the behavior of all build steps.
type BuildStepInterface interface {
	// Run the build step and wait, returns the result.
	Run() *BuildResult

	// Call right after the build step is executed only if it's successful.
	Finalize()
}

// Single build step that is part of a factory.
// This defines the properties shared by all build steps.
type BuildStep struct {
	BuildStepInterface
	// Factory that owns this build step.
	parent *Factory
	// Name of this build step.
	name string
	// Current working directory.
	cwd string
	// Summary with the most important log messages.
	summary map[string]string
	// Indicates whether a failure should keep the factory going.
	keepGoing bool
}

// Return whether the factory should keep going regardless a failure in this build step.
func (b *BuildStep) ShouldKeepGoing() bool {
	return b.keepGoing
}

// Execute a shell command and return results.
func (b *BuildStep) ExecShellCommand(name string, args []string, env []string) *BuildResult {
	result := &BuildResult{"", "", false, 0}

	// Prepare command
	cmd := exec.Command(name)
	cmd.Args = args
	cmd.Env = env

	// Capture output
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	err := cmd.Start()
	if err != nil {
		return result
	}

	// Measure elapsed time
	ticker := time.NewTicker(time.Second)
	go func(ticker *time.Ticker, result *BuildResult) {
		now := time.Now()
		for _ = range ticker.C {
			result.elapsedTime = time.Since(now)
		}
	}(ticker, result)

	// Kill processes that take too much time
	timer := time.NewTimer(4 * time.Hour)
	go func(timer *time.Timer, ticker *time.Ticker, cmd *exec.Cmd, result *BuildResult) {
		for _ = range timer.C {
			_ = cmd.Process.Signal(os.Kill)
			result.success = false
			ticker.Stop()
		}
	}(timer, ticker, cmd, result)

	// Wait for the process to return
	err = cmd.Wait()
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	result.success = err == nil

	return result
}

// Add a summary entry that will be shown by the user interface.
func (b *BuildStep) AddSummary(label string, value string) {
	b.summary[label] = value
}
