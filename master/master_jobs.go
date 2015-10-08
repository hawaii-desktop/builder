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
	"github.com/hawaii-desktop/builder/logging"
)

// Append a job to the list of pending jobs.
func (m *Master) appendJob(j *Job) {
	// Serialize actions on jobs slice
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	// Append job
	m.jobs = append(m.jobs, j)
}

// Remove a job from the list of pending jobs.
func (m *Master) removeJob(j *Job) {
	// Serialize actions on jobs slice
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	// Find index
	i := -1
	for k, v := range m.jobs {
		if v == j {
			i = k
			break
		}
	}
	if i == -1 {
		logging.Warningf("Unable to remove job #%d from the list\n", j.Id)
		return
	}

	// Remove
	m.jobs, m.jobs[len(m.jobs)-1] =
		append(m.jobs[:i], m.jobs[i+1:]...), nil
}

// Iterate the jobs list and execute the function on each of them.
func (m *Master) forEachJob(f func(j *Job)) {
	// Serialize actions on jobs slice
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	// Iterate
	for _, j := range m.jobs {
		f(j)
	}
}

// Queue a job.
func (m *Master) queueJob(j *Job) {
	// Update Web socket clients
	m.updateStatistics()
	m.updateAllJobs()

	// Push it onto the queue
	m.buildJobQueue <- j
	logging.Infof("Queued job #%d (target \"%s\" for %s)\n", j.Id, j.Target, j.Architecture)
}
