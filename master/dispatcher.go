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

package master

import (
	"github.com/hawaii-desktop/builder/common/logging"
	"time"
)

// Buffered channel that holds the job channels from each slave.
var SlaveQueue chan chan *Job

// Start the dispatcher of jobs to slaves.
func StartDispatcher() {
	// Initialize the channel
	SlaveQueue = make(chan chan *Job, Config.Build.MaxSlaves)

	go func() {
		for {
			select {
			case j := <-BuildJobQueue:
				logging.Tracef("About to dispatch job #%d...\n", j.Id)
				go func() {
					// Update job
					j.Started = time.Now()
					j.Status = JOB_STATUS_WAITING

					// Dispatch
					slave := <-SlaveQueue
					logging.Tracef("Dispatching job #%d...\n", j.Id)
					slave <- j
				}()
			}
		}
	}()
}
