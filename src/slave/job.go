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
	"github.com/hawaii-desktop/builder/src/logging"
)

// Package information for a build.
type PackageInfo struct {
	Ci                bool
	VcsUrl            string
	VcsBranch         string
	UpstreamVcsUrl    string
	UpstreamVcsBranch string
}

// Image information for a build.
type ImageInfo struct {
	VcsUrl    string
	VcsBranch string
}

// Describe a target.
type TargetInfo struct {
	Name         string
	Architecture string
	Package      *PackageInfo
	Image        *ImageInfo
}

// Represents a job.
type Job struct {
	// Identifier.
	Id uint64
	// Target information.
	Target *TargetInfo
	// Status.
	Status JobStatus
	// Channel used to signal when an update should be sent to master.
	UpdateChannel chan bool
	// Channel used to quit the goroutine responsible for sending updates to the master.
	CloseChannel chan bool
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
	JOB_STATUS_SUCCESSFUL:   "Successful",
	JOB_STATUS_FAILED:       "Failed",
	JOB_STATUS_CRASHED:      "Crashed",
}

// Create a new job object.
func NewJob(id uint64, t *TargetInfo) *Job {
	j := &Job{
		Id:            id,
		Target:        t,
		Status:        JOB_STATUS_WAITING,
		UpdateChannel: make(chan bool),
		CloseChannel:  make(chan bool),
	}
	return j
}

// Process the job.
func (j *Job) Process() {
	// Update job on master
	j.Status = JOB_STATUS_PROCESSING
	j.UpdateChannel <- true

	// Stop processing and log status on return
	defer func() {
		logging.Infof("Finished job #%d (target \"%s\" for %s) with status \"%s\"\n",
			j.Id, j.Target.Name, j.Target.Architecture, jobStatusDescriptionMap[j.Status])
		j.CloseChannel <- true
	}()

	// Log some information
	logging.Infof("Building job #%d (target \"%s\" for %s)...",
		j.Id, j.Target.Name, j.Target.Architecture)

	// Create a factory
	var f *Factory
	if j.Target.Package != nil {
		f = NewRpmFactory(j)
	} else if j.Target.Image != nil {
		f = NewImageFactory(j)
	} else {
		// We shouldn't reach here but I'm paranoid
		j.Status = JOB_STATUS_FAILED
		j.UpdateChannel <- true
		return
	}

	// Run factory
	if f.Run() {
		j.Status = JOB_STATUS_SUCCESSFUL
	} else {
		j.Status = JOB_STATUS_FAILED
	}
	f.Close()

	// Update job on master
	j.UpdateChannel <- true
}
