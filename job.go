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

package builder

import (
	"sync"
	"time"
)

// Represents a job.
type Job struct {
	// Identifier.
	Id uint64 `json:"id"`
	// Type.
	Type JobTargetType `json:"type"`
	// Target name.
	Target string `json:"target"`
	// Architecture.
	Architecture string `json:"arch"`
	// When the job has started.
	Started time.Time `json:"started"`
	// When the job has finished.
	Finished time.Time `json:"finished"`
	// Status.
	Status JobStatus `json:"status"`
	// Build steps.
	Steps []*Step `json:"steps"`
	// Mutex that serialize access to this job.
	Mutex sync.Mutex `json:"-"`
}

// Step represents the step of a job.
type Step struct {
	// Name.
	Name string `json:"name"`
	// When the step was started.
	Started time.Time `json:"started"`
	// When the step has finished.
	Finished time.Time `json:"finished"`
	// Summary.
	Summary map[string][]string `json:"summary"`
	// Output.
	Log string `json:"log"`
}

// Job target type enumeration.
type JobTargetType uint32

const (
	JOB_TARGET_TYPE_PACKAGE = iota
	JOB_TARGET_TYPE_IMAGE
)

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
var JobStatusDescriptionMap = map[JobStatus]string{
	JOB_STATUS_JUST_CREATED: "JustCreated",
	JOB_STATUS_WAITING:      "Waiting",
	JOB_STATUS_PROCESSING:   "Processing",
	JOB_STATUS_SUCCESSFUL:   "Successful",
	JOB_STATUS_FAILED:       "Failed",
	JOB_STATUS_CRASHED:      "Crashed",
}
