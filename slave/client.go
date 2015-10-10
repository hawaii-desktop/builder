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
	pb "github.com/hawaii-desktop/builder/protocol"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"runtime"
)

// Store important data used during the life time of the slave.
type Client struct {
	// RPC proxy.
	client pb.BuilderClient
	// Identifier for this slave, attributed after subscription.
	// Its value is 0 when unsubscribed.
	slaveId uint64
	// Channel for job processing.
	jobQueue chan *Job
	// Channel used to synchronize all goroutines.
	quit chan bool
}

// Map to encode job status.
var jobStatusMap = map[builder.JobStatus]pb.EnumJobStatus{
	builder.JOB_STATUS_JUST_CREATED: pb.EnumJobStatus_JOB_STATUS_JUST_CREATED,
	builder.JOB_STATUS_WAITING:      pb.EnumJobStatus_JOB_STATUS_WAITING,
	builder.JOB_STATUS_PROCESSING:   pb.EnumJobStatus_JOB_STATUS_PROCESSING,
	builder.JOB_STATUS_SUCCESSFUL:   pb.EnumJobStatus_JOB_STATUS_SUCCESSFUL,
	builder.JOB_STATUS_FAILED:       pb.EnumJobStatus_JOB_STATUS_FAILED,
	builder.JOB_STATUS_CRASHED:      pb.EnumJobStatus_JOB_STATUS_CRASHED,
}

// Create a new Client from a gRPC connection.
func NewClient(conn *grpc.ClientConn) *Client {
	// Create a RPC proxy and a queue for (NCPU/2)+1 jobs to be
	// processed at the same time
	c := &Client{}
	c.client = pb.NewBuilderClient(conn)
	c.slaveId = 0
	c.jobQueue = make(chan *Job, (runtime.NumCPU()/2)+1)
	c.quit = make(chan bool)

	// Process jobs as soon as they are dispatched to us
	go func() {
		for {
			select {
			case j := <-c.jobQueue:
				j.Process()
			case <-c.quit:
				return
			}
		}
	}()

	return c
}

// Close connection with the master and exit all goroutines.
func (c *Client) Close() {
	c.quit <- true
	close(c.jobQueue)
	close(c.quit)
	c.client = nil
}

// Subscribe to the master.
func (c *Client) Subscribe() error {
	// Subscribe and take the stream
	stream, err := c.client.Subscribe(context.Background())
	if err != nil {
		return err
	}

	// Function that send job updates back to the master
	var sendJobUpdate = func(j *Job) {
		args := &pb.InputMessage{
			Payload: &pb.InputMessage_JobUpdate{
				JobUpdate: &pb.JobUpdateRequest{
					SlaveId: c.slaveId,
					Id:      j.Id,
					Status:  jobStatusMap[j.Status],
				},
			},
		}
		stream.Send(args)
	}

	// Function that send job updates back to the master
	var sendStepUpdate = func(j *Job, bs *BuildStep) {
		args := &pb.InputMessage{
			Payload: &pb.InputMessage_StepUpdate{
				StepUpdate: &pb.StepResponse{
					JobId:    j.Id,
					Name:     bs.Name,
					Running:  !bs.finished.IsZero(),
					Started:  bs.started.UnixNano(),
					Finished: bs.finished.UnixNano(),
					Log:      bs.output,
				},
			},
		}
		stream.Send(args)
	}

	// Read from stream
	wait := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(wait)
				return
			}
			if err != nil {
				logging.Errorf("Failed to receive stream: %s", err)
				return
			}

			// Subscription reply
			subscription := in.GetSubscription()
			if subscription != nil {
				c.slaveId = subscription.Id
				logging.Infof("Slave subscribed with id %d\n", c.slaveId)
			}

			// Job dispatched to us
			jobDispatch := in.GetJobDispatch()
			if jobDispatch != nil {
				// Read build information from the request
				var (
					target string
					arch   string
				)
				pkg := jobDispatch.GetPackage()
				if pkg != nil {
					target = pkg.Name
					arch = pkg.Architectures[0]
				}
				img := jobDispatch.GetImage()
				if img != nil {
					target = img.Name
					arch = img.Architectures[0]
				}

				// Create a new job
				logging.Infof("Processing job #%d (target \"%s\" for %s)\n",
					jobDispatch.Id, target, arch)
				var pkgInfo *PackageInfo = nil
				var imgInfo *ImageInfo = nil
				if pkg != nil {
					pkgInfo = &PackageInfo{
						Ci:                pkg.Ci,
						VcsUrl:            pkg.Vcs.Url,
						VcsBranch:         pkg.Vcs.Branch,
						UpstreamVcsUrl:    pkg.UpstreamVcs.Url,
						UpstreamVcsBranch: pkg.UpstreamVcs.Branch,
					}
				} else if img != nil {
					imgInfo = &ImageInfo{
						VcsUrl:    img.Vcs.Url,
						VcsBranch: img.Vcs.Branch,
					}
				}
				j := NewJob(jobDispatch.Id, target, arch, &TargetInfo{pkgInfo, imgInfo})

				// Send updates back to master
				go func() {
					for {
						select {
						case <-j.UpdateChannel:
							sendJobUpdate(j)
							break
						case bs := <-j.stepUpdateQueue:
							sendStepUpdate(j, bs)
							break
						case <-j.CloseChannel:
							j = nil
							return
						}
					}
				}()

				// Process
				c.jobQueue <- j
			}
		}
	}()

	// First of all: subscribe
	args := &pb.InputMessage{
		Payload: &pb.InputMessage_Subscription{
			Subscription: &pb.SubscribeRequest{
				Name:          Config.Slave.Name,
				Channels:      Config.Slave.Channels,
				Architectures: Config.Slave.Architectures,
			},
		},
	}
	stream.Send(args)

	// Wait until the stream is clsed
	<-wait
	stream.CloseSend()

	return nil
}

// Unsubscribe from the master.
func (c *Client) Unsubscribe() error {
	args := &pb.UnsubscribeRequest{Id: c.slaveId}
	reply, err := c.client.Unsubscribe(context.Background(), args)
	if reply.Result {
		c.slaveId = 0
	}
	return err
}
