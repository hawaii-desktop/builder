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
	"errors"
	"fmt"
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/database"
	"github.com/hawaii-desktop/builder/logging"
	pb "github.com/hawaii-desktop/builder/protocol"
	"github.com/hawaii-desktop/builder/utils"
	"golang.org/x/net/context"
	"io"
	"regexp"
	"sync"
	"time"
)

// RPC service.
type RpcService struct {
	// List of slaves.
	Slaves []*Slave
	// Protects slave list.
	sMutex sync.Mutex
	// Master.
	master *Master
}

// Errors
var (
	ErrAlreadySubscribed  = errors.New("slave has already subscribed")
	ErrSlaveNotFound      = errors.New("slave not found")
	ErrInvalidSlave       = errors.New("slave is not valid")
	ErrJobNotFound        = errors.New("job not found with that id")
	ErrNoMatchingPackages = errors.New("no matching packages")
	ErrNoMatchingImages   = errors.New("no matching images")
)

// Map to decode job type.
var jobTargetMap = map[pb.EnumTargetType]builder.JobTargetType{
	pb.EnumTargetType_PACKAGE: builder.JOB_TARGET_TYPE_PACKAGE,
	pb.EnumTargetType_IMAGE:   builder.JOB_TARGET_TYPE_IMAGE,
}

// Map to decode job status.
var jobStatusMap = map[pb.EnumJobStatus]builder.JobStatus{
	pb.EnumJobStatus_JOB_STATUS_JUST_CREATED: builder.JOB_STATUS_JUST_CREATED,
	pb.EnumJobStatus_JOB_STATUS_WAITING:      builder.JOB_STATUS_WAITING,
	pb.EnumJobStatus_JOB_STATUS_PROCESSING:   builder.JOB_STATUS_PROCESSING,
	pb.EnumJobStatus_JOB_STATUS_SUCCESSFUL:   builder.JOB_STATUS_SUCCESSFUL,
	pb.EnumJobStatus_JOB_STATUS_FAILED:       builder.JOB_STATUS_FAILED,
	pb.EnumJobStatus_JOB_STATUS_CRASHED:      builder.JOB_STATUS_CRASHED,
}

// Allocate a new RpcService with an empty list of slaves.
// The slaves list is initially empty and has a capacity as big as the maximum
// number of slaves from the configuration.
// The jobs list is initially empty and has a capacity as big as
// the maximum number of jobs from the configuration.
func NewRpcService(master *Master) *RpcService {
	return &RpcService{
		Slaves: make([]*Slave, 0, Config.Build.MaxSlaves),
		master: master,
	}
}

// Slave call this to initiate a dialog.
func (m *RpcService) Subscribe(stream pb.Builder_SubscribeServer) error {
	// Function to send back the slave identifier
	var sendSlaveId = func(s *Slave) {
		response := &pb.SubscribeResponse{Id: s.Id}
		reply := &pb.OutputMessage{
			Payload: &pb.OutputMessage_Subscription{
				Subscription: response,
			},
		}
		stream.Send(reply)
	}

	// Function to send a job to a slave
	var sendJob = func(s *Slave, j *Job) {
		// Helper function that actually sends the dispatch request
		var sendEnvelope = func(response *pb.JobDispatchRequest) {
			reply := &pb.OutputMessage{
				Payload: &pb.OutputMessage_JobDispatch{
					JobDispatch: response,
				},
			}
			stream.Send(reply)
			logging.Infof("Job #%d scheduled on \"%s\"\n", j.Id, s.Name)
		}

		// Retrieve target information and send
		switch j.Type {
		case builder.JOB_TARGET_TYPE_PACKAGE:
			pkg := m.master.db.GetPackage(j.Target)
			if pkg == nil {
				return
			}
			pkgmsg := &pb.PackageInfo{
				Name:          pkg.Name,
				Architectures: []string{j.Architecture},
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
			response := &pb.JobDispatchRequest{
				Id: j.Id,
				Payload: &pb.JobDispatchRequest_Package{
					Package: pkgmsg,
				},
			}
			sendEnvelope(response)
			break
		case builder.JOB_TARGET_TYPE_IMAGE:
			img := m.master.db.GetImage(j.Target)
			if img == nil {
				return
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
			response := &pb.JobDispatchRequest{
				Id: j.Id,
				Payload: &pb.JobDispatchRequest_Image{
					Image: imgmsg,
				},
			}
			sendEnvelope(response)
			break
		}
	}

	// Slave loop
	var slaveLoop = func(s *Slave) {
		for _, topic := range s.Topics() {
			go func(topic string) {
				for {
					// Do not queue a slave that suddenly unregisters itself
					if !s.Subscribed || !s.Active {
						return
					}

					// Add to the queue
					m.master.slaveQueues[topic] <- s.jobChannels[topic]

					select {
					case j := <-s.jobChannels[topic]:
						// Send the job to the slave
						sendJob(s, j)

						// Wait for processing on the other side
						<-j.Channel
					case <-s.quitChannels[topic]:
						// Slave has been asked to stop
						return
					}
				}
			}(topic)
		}
	}

	for {
		// Read request from the stream
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		subscription := in.GetSubscription()
		if subscription != nil {
			// Register a new slave
			s, err := m.createSlave(subscription)
			if err != nil {
				return err
			}

			// Send back the identifier
			sendSlaveId(s)

			// Start dispatching to this slave
			go slaveLoop(s)
		}

		jobUpdate := in.GetJobUpdate()
		if jobUpdate != nil {
			var slave *Slave = nil
			m.sMutex.Lock()
			for _, s := range m.Slaves {
				if s.Id == jobUpdate.SlaveId {
					slave = s
					break
				}
			}
			m.sMutex.Unlock()

			var j *Job = nil
			m.master.forEachJob(func(curJob *Job) {
				// Skip other jobs
				if curJob.Id == jobUpdate.Id {
					j = curJob
				}
			})
			if j == nil {
				logging.Errorf("Cannot find job #%d\n", jobUpdate.Id)
				return ErrJobNotFound
			}

			// Update the status
			j.Status = jobStatusMap[jobUpdate.Status]

			// Handle status change
			if j.Status >= builder.JOB_STATUS_SUCCESSFUL && j.Status <= builder.JOB_STATUS_CRASHED {
				// Update finished time and notify
				j.Finished = time.Now()

				// Log the status
				if j.Status == builder.JOB_STATUS_SUCCESSFUL {
					logging.Infof("Job #%d completed successfully on \"%s\"\n",
						j.Id, slave.Name)
				} else {
					logging.Errorf("Job #%d failed on \"%s\"\n",
						j.Id, slave.Name)
				}

				// Remove from the list
				m.master.removeJob(j)

				// Proceed to the next job
				j.Channel <- true
			} else {
				logging.Tracef("Change job #%d status to \"%s\"\n",
					jobUpdate.Id, builder.JobStatusDescriptionMap[j.Status])
			}

			// Save on the database
			m.master.saveDatabaseJob(j)

			// Update Web socket clients
			m.master.updateStatistics()
			m.master.updateAllJobs()
		}

		stepUpdate := in.GetStepUpdate()
		if stepUpdate != nil {
			// Do we have a valid job here?
			var j *Job = nil
			m.master.forEachJob(func(curJob *Job) {
				// Skip other jobs
				if curJob.Id == stepUpdate.JobId {
					j = curJob
				}
			})
			if j == nil {
				logging.Errorf("Cannot find job #%d\n", stepUpdate.JobId)
				return ErrJobNotFound
			}

			// Append or replace the build step
			j.Mutex.Lock()
			found := false
			for _, step := range j.Steps {
				if step.Name == stepUpdate.Name {
					step.Started = time.Unix(0, stepUpdate.Started)
					step.Finished = time.Unix(0, stepUpdate.Finished)
					step.Summary = utils.MapSliceString(stepUpdate.Summary)
					step.Log = string(stepUpdate.Log)
					step.Logs = stepUpdate.Logs
					found = true
					break
				}
			}
			if !found {
				step := &builder.Step{
					Name:     stepUpdate.Name,
					Started:  time.Unix(0, stepUpdate.Started),
					Finished: time.Unix(0, stepUpdate.Finished),
					Summary:  utils.MapSliceString(stepUpdate.Summary),
					Log:      string(stepUpdate.Log),
					Logs:     stepUpdate.Logs,
				}
				j.Steps = append(j.Steps, step)
			}
			j.Mutex.Unlock()

			// Save on the database
			m.master.saveDatabaseJob(j)

			// Update Web socket clients
			m.master.updateStatistics()
			m.master.updateAllJobs()
		}
	}
}

// Unregister a slave and stop it immediately.
// Jobs will not be dispatched to this slave until it has subscribed again.
func (m *RpcService) Unsubscribe(ctx context.Context, args *pb.UnsubscribeRequest) (*pb.UnsubscribeResponse, error) {
	// Prepare response
	reply := &pb.UnsubscribeResponse{Result: false}

	// Find slave and unregister
	m.sMutex.Lock()
	defer m.sMutex.Unlock()
	for _, slave := range m.Slaves {
		if slave == nil {
			continue
		}
		if slave.Id == args.Id {
			logging.Infof("Slave \"%s\" unsubscribed", slave.Name)
			slave.Subscribed = false
			slave.Stop()
			reply.Result = true
			return reply, nil
		}
	}

	// Reply
	reply.Result = false
	return reply, ErrSlaveNotFound
}

// Create and enqueue a job.
func (m *RpcService) CollectJob(ctx context.Context, args *pb.CollectJobRequest) (*pb.CollectJobResponse, error) {
	var (
		result bool   = false
		id     uint64 = 0
	)

	j, err := m.enqueueJob(args.Target, args.Architecture, args.Type)
	if err != nil {
		return nil, err
	}
	result = j != nil
	if j != nil {
		id = j.Id
	}

	reply := &pb.CollectJobResponse{Result: result, Id: id}
	return reply, nil
}

// Add or update a package.
func (m *RpcService) AddPackage(ctx context.Context, args *pb.PackageInfo) (*pb.BooleanMessage, error) {
	pkg := &database.Package{
		Name:          args.Name,
		Architectures: args.Architectures,
		Ci:            args.Ci,
		Vcs: database.VcsInfo{
			Url:    args.Vcs.Url,
			Branch: args.Vcs.Branch,
		},
		UpstreamVcs: database.VcsInfo{
			Url:    args.UpstreamVcs.Url,
			Branch: args.UpstreamVcs.Branch,
		},
	}
	if err := m.master.db.AddPackage(pkg); err != nil {
		return nil, err
	}
	return &pb.BooleanMessage{Result: true}, nil
}

// Remove package.
func (m *RpcService) RemovePackage(ctx context.Context, args *pb.StringMessage) (*pb.BooleanMessage, error) {
	err := m.master.db.RemovePackage(args.Name)
	if err != nil {
		return nil, err
	}
	return &pb.BooleanMessage{Result: true}, nil
}

// List packages matching the regular expression.
func (m *RpcService) ListPackages(args *pb.StringMessage, stream pb.Builder_ListPackagesServer) error {
	r, err := regexp.Compile(args.Name)
	if err != nil {
		return err
	}

	list := m.master.db.ListAllPackages()
	if len(list) == 0 {
		return ErrNoMatchingPackages
	}

	for _, pkg := range list {
		if !r.MatchString(pkg.Name) {
			continue
		}
		reply := &pb.PackageInfo{
			Name:          pkg.Name,
			Architectures: pkg.Architectures,
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
		stream.Send(reply)
	}

	return nil
}

// Add or update an image.
func (m *RpcService) AddImage(ctx context.Context, args *pb.ImageInfo) (*pb.BooleanMessage, error) {
	img := &database.Image{
		Name:          args.Name,
		Description:   args.Description,
		Architectures: args.Architectures,
		Vcs: database.VcsInfo{
			Url:    args.Vcs.Url,
			Branch: args.Vcs.Branch,
		},
	}
	if err := m.master.db.AddImage(img); err != nil {
		return nil, err
	}
	return &pb.BooleanMessage{Result: true}, nil
}

// Remove an image.
func (m *RpcService) RemoveImage(ctx context.Context, args *pb.StringMessage) (*pb.BooleanMessage, error) {
	err := m.master.db.RemoveImage(args.Name)
	if err != nil {
		return nil, err
	}
	return &pb.BooleanMessage{Result: true}, nil
}

// List images matching the regular expression.
func (m *RpcService) ListImages(args *pb.StringMessage, stream pb.Builder_ListImagesServer) error {
	r, err := regexp.Compile(args.Name)
	if err != nil {
		return err
	}

	list := m.master.db.ListAllImages()
	if len(list) == 0 {
		return ErrNoMatchingImages
	}

	for _, img := range list {
		if !r.MatchString(img.Name) {
			continue
		}
		reply := &pb.ImageInfo{
			Name:          img.Name,
			Description:   img.Description,
			Architectures: img.Architectures,
			Vcs: &pb.VcsInfo{
				Url:    img.Vcs.Url,
				Branch: img.Vcs.Branch,
			},
		}
		stream.Send(reply)
	}

	return nil
}

// Create a slave from the subscription request.
func (m *RpcService) createSlave(args *pb.SubscribeRequest) (*Slave, error) {
	// The same slave cannot subscribe twice
	m.sMutex.Lock()
	defer m.sMutex.Unlock()
	for _, slave := range m.Slaves {
		if slave == nil {
			continue
		}
		if slave.Name == args.Name && slave.Subscribed {
			return nil, ErrAlreadySubscribed
		}
	}

	// Create and append slave
	slave := NewSlave(m.master.db.NewSlaveId(), args.Name, args.Types, args.Architectures)
	m.Slaves = append(m.Slaves, slave)
	logging.Infof("Subscribed slave \"%s\" with id %d\n", slave.Name, slave.Id)
	return slave, nil
}

// Enqueue a job.
func (m *RpcService) enqueueJob(target, arch string, t pb.EnumTargetType) (*Job, error) {
	// Verify if the target exists
	switch t {
	case pb.EnumTargetType_PACKAGE:
		if !m.master.db.HasPackage(target) {
			return nil, fmt.Errorf("%s package not found", target)
		}
		break
	case pb.EnumTargetType_IMAGE:
		if !m.master.db.HasImage(target) {
			return nil, fmt.Errorf("%s image not found", target)
		}
		break
	default:
		return nil, fmt.Errorf("Wrong target type specified for \"%s\" (%s)\n", target, arch)
	}

	// Create a job
	j := &Job{
		&builder.Job{
			Id:           m.master.db.NewJobId(),
			Type:         jobTargetMap[t],
			Target:       target,
			Architecture: arch,
			Started:      time.Now(),
			Finished:     time.Time{},
			Status:       builder.JOB_STATUS_JUST_CREATED,
			Steps:        make([]*builder.Step, 0),
		},
		make(chan bool),
	}

	// Append job
	m.master.appendJob(j)

	// Save on the database
	m.master.saveDatabaseJob(j)

	// Push it onto the queue
	m.master.queueJob(j)

	return j, nil
}
