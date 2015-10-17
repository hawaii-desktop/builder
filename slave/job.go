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
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/logging"
	"golang.org/x/net/context"
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

// Image information for a build.
type ImageInfo struct {
	VcsUrl    string
	VcsBranch string
}

// Describe a target.
type TargetInfo struct {
	Package *PackageInfo
	Image   *ImageInfo
}

// Represents a job.
type Job struct {
	// Base.
	*builder.Job
	// Target information.
	Info *TargetInfo
	// Channel used to signal when an update should be sent to master.
	UpdateChannel chan bool
	// Context.
	ctx context.Context
	// Build step updates are queued here and then sent to the master.
	stepUpdateQueue chan *BuildStep
	// Channel used to quit the goroutine responsible for sending updates to the master.
	CloseChannel chan bool
	// Artifacts.
	artifacts []*Artifact
	// Send a value to this channel to trigger artifacts upload.
	artifactsChannel chan bool
}

// Artifact.
type Artifact struct {
	// Artifact full path on slave.
	Source string
	// Artifact full path on master.
	Destination string
	// File permission on master.
	Permission uint32
}

// Create a new job object.
func NewJob(ctx context.Context, id uint64, target, arch string, info *TargetInfo) *Job {
	j := &Job{
		&builder.Job{Id: id,
			Target:       target,
			Architecture: arch,
			Started:      time.Time{},
			Finished:     time.Time{},
			Status:       builder.JOB_STATUS_WAITING,
		},
		info,
		make(chan bool),
		ctx,
		make(chan *BuildStep),
		make(chan bool),
		make([]*Artifact, 0),
		make(chan bool),
	}
	return j
}

// Process the job.
func (j *Job) Process() {
	// Update job on master
	j.Status = builder.JOB_STATUS_PROCESSING
	j.UpdateChannel <- true

	// Stop processing and log status on return
	defer func() {
		logging.Infof("Finished job #%d (target \"%s\" for %s) with status \"%s\"\n",
			j.Id, j.Target, j.Architecture, builder.JobStatusDescriptionMap[j.Status])
		j.CloseChannel <- true
	}()

	// Log some information
	logging.Infof("Building job #%d (target \"%s\" for %s)...",
		j.Id, j.Target, j.Architecture)

	// Create a factory
	var f *Factory
	if j.Info.Package != nil {
		f = NewRpmFactory(j)
	} else if j.Info.Image != nil {
		f = NewImageFactory(j)
	} else {
		// We shouldn't reach here but I'm paranoid
		j.Status = builder.JOB_STATUS_FAILED
		j.UpdateChannel <- true
		return
	}

	// Run factory
	if f.Run() {
		j.Status = builder.JOB_STATUS_SUCCESSFUL
	} else {
		j.Status = builder.JOB_STATUS_FAILED
	}

	// Close factory
	f.Close()

	// Upload artifacts
	if j.Status == builder.JOB_STATUS_SUCCESSFUL {
		j.artifactsChannel <- true
	}

	// Update job on master
	j.UpdateChannel <- true
}
