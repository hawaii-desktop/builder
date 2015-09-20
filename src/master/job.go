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
	"time"
)

// Holds the last global job identifier.
var globalJobId uint64 = 0

// Represents a job.
type Job struct {
	// Identifier.
	Id uint64
	// Target name.
	Target string
	// Architecture.
	Architecture string
	// When the job has started.
	Started time.Time
	// When the job has finished.
	Finished time.Time
	// Status.
	Status JobStatus
	// Channel.
	Channel chan bool
}

// Job status enumeration.
type JobStatus uint32

const (
	JOB_STATUS_JUST_CREATED = iota
	JOB_STATUS_WAITING
	JOB_STATUS_PROCESSING
	JOB_STATUS_SUCCESSFUL
	JOB_STATUS_FAILED
	JOB_STATUS_CRASHED
)

// Map job status to description.
var jobStatusDescriptionMap = map[JobStatus]string{
	JOB_STATUS_JUST_CREATED: "JustCreated",
	JOB_STATUS_WAITING:      "Waiting",
	JOB_STATUS_PROCESSING:   "Processing",
	JOB_STATUS_FAILED:       "Failed",
	JOB_STATUS_CRASHED:      "Crashed",
}
