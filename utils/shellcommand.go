/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
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

import (
	"os/exec"
	"time"
)

// Run a command with timeout.
func RunWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	t := time.AfterFunc(timeout, func() { cmd.Process.Kill() })
	defer t.Stop()
	return cmd.Wait()
}

// Run a command with timeout and return the combined output.
func RunCombinedWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	t := time.AfterFunc(timeout, func() { cmd.Process.Kill() })
	defer t.Stop()
	return cmd.CombinedOutput()
}
