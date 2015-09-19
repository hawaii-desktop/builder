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

package utils

// Execute a shell command and return the output.
func ExecShellCommand(name string, args []string, env []string) ([]byte, []byte, error) {
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

	// Run the command
	if err := cmd.Start(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}
	err = cmd.Wait()
	return stdout.Bytes(), stderr.Bytes(), err
}

// Execute a shell command with a timeout and return the output.
func ExecShellCommandWithTimeout(name string, args []string, env []string, timeout time.Duration) ([]byte, []byte, error) {
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

	// Run the command
	err := runWithTimeout(cmd, timeout)
	return stdout.Bytes(), stderr.Bytes(), err
}

// Run a command with timeout.
func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	t := time.AfterFunc(timeout, func() { cmd.Process.Kill() })
	defer t.Stop()
	return cmd.Wait()
}
