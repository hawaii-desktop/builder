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
	"time"
)

// Package information for a build.
type PackageInfo struct {
	Ci                bool
	VcsUrl            string
	VcsBranch         string
	UpstreamVcsUrl    string
	UpstreamVcsBranch string
}

// Represents a job.
type Job struct {
	// Identifier.
	Id uint64
	// Target name.
	Target string
	// Architecture.
	Architecture string
	// Package information.
	Package *PackageInfo
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
	JOB_STATUS_FAILED:       "Failed",
	JOB_STATUS_CRASHED:      "Crashed",
}

// Create a new job object.
func NewJob(id uint64, target string, arch string, pkg *PackageInfo) *Job {
	j := &Job{
		Id:            id,
		Target:        target,
		Architecture:  arch,
		Package:       pkg,
		Status:        JOB_STATUS_WAITING,
		UpdateChannel: make(chan bool),
		CloseChannel:  make(chan bool),
	}
	return j
}

// Process the job.
func (j *Job) Process() {
	// Processing
	j.Status = JOB_STATUS_PROCESSING
	j.UpdateChannel <- true

	// TODO: Fetch sources
	// TODO: Build
	logging.Infoln("...")
	time.Sleep(10 * time.Second)

	// Failed
	j.Status = JOB_STATUS_FAILED
	j.UpdateChannel <- true
	logging.Infof("Finished job #%d (target \"%s\") with status \"%s\"\n",
		j.Id, j.Target, jobStatusDescriptionMap[j.Status])

	// Stop processing
	j.CloseChannel <- true
}
