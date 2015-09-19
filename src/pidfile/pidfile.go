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

package pidfile

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type PidFile string

var (
	ErrNoAbsolutePath = errors.New("an absolute path must be given")
	ErrParseFailed    = errors.New("file cannot be parsed")
	ErrInvalidPid     = errors.New("file contains an invalid PID")
	ErrAlreadyLocked  = errors.New("another process is using this PID file")
	ErrUnableToLock   = errors.New("failed to lock PID file")
)

// Creates a new PidFile and an error.
// Path must be an absolute path, otherwise an error is returned.
func New(path string) (PidFile, error) {
	if !filepath.IsAbs(path) {
		return PidFile(""), ErrNoAbsolutePath
	}
	return PidFile(path), nil
}

// Returns the process that currently owns the PID file, or nil
// if the PID file is not used by any process.
func (l PidFile) CurrentOwner() (*os.Process, error) {
	// Retrieve the content of the PID file
	file, err := os.Open(string(l))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the PID from the file
	var pid int
	_, err = fmt.Fscanf(file, "%d\n", &pid)
	if err != nil {
		return nil, ErrInvalidPid
	}

	// Try finding the process
	if pid > 0 {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return nil, err
		}

		return proc, nil
	}

	return nil, nil
}

// Try locking the PID file, give up if another process is already using it.
// If a stale PID file is found and no valid process is claiming it, the
// PID file is taken over.
func (l PidFile) TryLock() error {
	// Is it claimed by another process?
	proc, err := l.CurrentOwner()
	if proc == nil {
		os.Remove(string(l))
	} else {
		if err != nil {
			return err
		}
		return ErrAlreadyLocked
	}

	// Create directories
	err = os.MkdirAll(filepath.Dir(string(l)), 0755)
	if err != nil {
		return err
	}

	// Create a temporary file
	file, err := ioutil.TempFile(filepath.Dir(string(l)), "")
	if err == nil {
		defer file.Close()
		defer os.Remove(file.Name())
	} else {
		return err
	}

	// Write the PID
	_, err = file.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	if err != nil {
		return err
	}

	// Link temporary file with PID file
	_ = os.Link(file.Name(), string(l))

	// Get information of both files
	tmp, err := os.Lstat(file.Name())
	if err != nil {
		return err
	}
	lock, err := os.Lstat(string(l))
	if err != nil {
		return err
	}

	// If they are the same we are ok
	if os.SameFile(tmp, lock) {
		return nil
	}

	return ErrUnableToLock
}

// Remove the PID file.
func (l PidFile) Unlock() error {
	return os.Remove(string(l))
}
