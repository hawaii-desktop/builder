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

package master

import (
	"github.com/hawaii-desktop/builder"
	pb "github.com/hawaii-desktop/builder/protocol"
)

// dispatchSlave starts job dispatching to slave.
func (m *Master) dispatchSlave(slave *Slave, channel chan<- *pb.JobRequest) {
	// Start dispatching to this slave
	for _, topic := range slave.Topics() {
		go func(topic string) {
			for {
				// Do not queue a slave that suddenly unregisters itself
				if !slave.Subscribed || !slave.Active {
					return
				}

				// Add to the queue
				m.slaveQueues[topic] <- slave.jobChannels[topic]

				select {
				case job := <-slave.jobChannels[topic]:
					// Send the job to the slave
					r := m.sendJobToSlave(slave, job)
					if r != nil {
						channel <- r
					}

					// Wait for processing on the other side
					<-job.Channel
				case <-slave.quitChannels[topic]:
					// Slave has been asked to stop
					return
				}
			}
		}(topic)
	}
}

// sendJobToSlave dispatches a job to slave.
func (m *Master) sendJobToSlave(slave *Slave, job *Job) *pb.JobRequest {
	// Retrieve target information and send
	switch job.Type {
	case builder.JOB_TARGET_TYPE_PACKAGE:
		pkg := m.db.GetPackage(job.Target)
		if pkg == nil {
			return nil
		}
		pkgmsg := &pb.PackageInfo{
			Name:          pkg.Name,
			Architectures: []string{job.Architecture},
			Ci:            pkg.Ci,
			Vcs: &pb.VcsInfo{
				Url:    pkg.Vcs.Url,
				Branch: pkg.Vcs.Branch,
			},
			UpstreamVcs: &pb.VcsInfo{
				Url:    pkg.UpstreamVcs.Url,
				Branch: pkg.UpstreamVcs.Branch,
			},
		}
		return &pb.JobRequest{
			Id: job.Id,
			Payload: &pb.JobRequest_Package{
				Package: pkgmsg,
			},
		}
	case builder.JOB_TARGET_TYPE_IMAGE:
		img := m.db.GetImage(job.Target)
		if img == nil {
			return nil
		}
		imgmsg := &pb.ImageInfo{
			Name:          img.Name,
			Description:   img.Description,
			Architectures: img.Architectures,
			Vcs: &pb.VcsInfo{
				Url:    img.Vcs.Url,
				Branch: img.Vcs.Branch,
			},
		}
		return &pb.JobRequest{
			Id: job.Id,
			Payload: &pb.JobRequest_Image{
				Image: imgmsg,
			},
		}
	}

	return nil
}
