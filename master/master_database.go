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
	"github.com/hawaii-desktop/builder"
)

// Load jobs that were created and never dispatched before.
// Jobs could have been created and then the master could have been
// shut down before any slave could process them.
func (m *Master) LoadDatabaseJobs() {
	m.db.ForEachJob(func(job *builder.Job) {
		if job.Status != builder.JOB_STATUS_JUST_CREATED &&
			job.Status != builder.JOB_STATUS_WAITING {
			return
		}

		j := &Job{
			&builder.Job{
				Id:           job.Id,
				Type:         job.Type,
				Target:       job.Target,
				Architecture: job.Architecture,
				Started:      job.Started,
				Finished:     job.Finished,
				Status:       job.Status,
			},
			make(chan bool),
		}
		m.appendJob(j)
		m.queueJob(j)
	})
}

// Save job on the database.
func (m *Master) saveDatabaseJob(job *Job) {
	m.db.SaveJob(job)
}
