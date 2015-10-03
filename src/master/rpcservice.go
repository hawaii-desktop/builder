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

package main

import (
	"errors"
	"fmt"
	"github.com/hawaii-desktop/builder/src/api"
	"github.com/hawaii-desktop/builder/src/database"
	"github.com/hawaii-desktop/builder/src/logging"
	pb "github.com/hawaii-desktop/builder/src/protocol"
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
	// List of jobs to be processed.
	Jobs []*Job
	// Protects slave list.
	sMutex sync.Mutex
	// Protects jobs list.
	jMutex sync.Mutex
	// Database.
	db *database.Database
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
var jobTargetMap = map[pb.EnumTargetType]JobTargetType{
	pb.EnumTargetType_PACKAGE: JOB_TARGET_TYPE_PACKAGE,
	pb.EnumTargetType_IMAGE:   JOB_TARGET_TYPE_IMAGE,
}

// Map to decode job status.
var jobStatusMap = map[pb.EnumJobStatus]api.JobStatus{
	pb.EnumJobStatus_JOB_STATUS_JUST_CREATED: api.JOB_STATUS_JUST_CREATED,
	pb.EnumJobStatus_JOB_STATUS_WAITING:      api.JOB_STATUS_WAITING,
	pb.EnumJobStatus_JOB_STATUS_PROCESSING:   api.JOB_STATUS_PROCESSING,
	pb.EnumJobStatus_JOB_STATUS_SUCCESSFUL:   api.JOB_STATUS_SUCCESSFUL,
	pb.EnumJobStatus_JOB_STATUS_FAILED:       api.JOB_STATUS_FAILED,
	pb.EnumJobStatus_JOB_STATUS_CRASHED:      api.JOB_STATUS_CRASHED,
}

// Allocate a new RpcService with an empty list of slaves.
// The slaves list is initially empty and has a capacity as big as the maximum
// number of slaves from the configuration.
// The jobs list is initially empty and has a capacity as big as
// the maximum number of jobs from the configuration.
// This also create or open the database.
func NewRpcService(master *Master) (*RpcService, error) {
	db, err := database.NewDatabase(Config.Server.Database)
	if err != nil {
		return nil, err
	}

	m := &RpcService{}
	m.Slaves = make([]*Slave, 0, Config.Build.MaxSlaves)
	m.Jobs = make([]*Job, 0, Config.Build.MaxJobs)
	m.db = db
	m.master = master
	return m, nil
}

// Close the database.
func (m *RpcService) Close() {
	m.db.Close()
	m.db = nil
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
		case JOB_TARGET_TYPE_PACKAGE:
			pkg := m.db.GetPackage(j.Target)
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
		case JOB_TARGET_TYPE_IMAGE:
			img := m.db.GetImage(j.Target)
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
		for {
			// Do not queue a slave that suddenly unregisters itself
			if !s.Subscribed || !s.Active {
				return
			}

			// Add to the queue
			m.master.slaveQueue <- s.JobChannel

			select {
			case j := <-s.JobChannel:
				// Send the job to the slave
				sendJob(s, j)

				// Wait for processing on the other side
				<-j.Channel
			case <-s.QuitChannel:
				// Slave has been asked to stop
				return
			}
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

			m.jMutex.Lock()
			var j *Job = nil
			for _, curJob := range m.Jobs {
				// Skip other jobs
				if curJob.Id == jobUpdate.Id {
					j = curJob
				}
			}
			if j == nil {
				logging.Errorf("Cannot find job #%d\n", jobUpdate.Id)
				return ErrJobNotFound
			}
			m.jMutex.Unlock()

			// Update the status
			j.Status = jobStatusMap[jobUpdate.Status]

			// Handle finished jobs
			if j.Status >= api.JOB_STATUS_SUCCESSFUL && j.Status <= api.JOB_STATUS_CRASHED {
				// Update finished time and notify
				j.Finished = time.Now()

				// Log the status
				if j.Status == api.JOB_STATUS_SUCCESSFUL {
					logging.Infof("Job #%d completed successfully on \"%s\"\n",
						j.Id, slave.Name)
				} else {
					logging.Errorf("Job #%d failed on \"%s\"\n",
						j.Id, slave.Name)
				}

				// Save on the database
				dbJob := &database.Job{j.Id, j.Target, j.Architecture, j.Started, j.Finished, j.Status}
				m.db.SaveJob(dbJob)
				dbJob = nil

				// Remove from the list
				m.removeJob(j)

				// Broadcast web clients
				m.calculateStats()

				// Proceed to the next job
				j.Channel <- true
			} else {
				logging.Tracef("Change job #%d status to \"%s\"\n",
					jobUpdate.Id, api.JobStatusDescriptionMap[j.Status])
			}
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
	if err := m.db.AddPackage(pkg); err != nil {
		return nil, err
	}
	return &pb.BooleanMessage{Result: true}, nil
}

// Remove package.
func (m *RpcService) RemovePackage(ctx context.Context, args *pb.StringMessage) (*pb.BooleanMessage, error) {
	err := m.db.RemovePackage(args.Name)
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

	list := m.db.ListAllPackages()
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
	if err := m.db.AddImage(img); err != nil {
		return nil, err
	}
	return &pb.BooleanMessage{Result: true}, nil
}

// Remove an image.
func (m *RpcService) RemoveImage(ctx context.Context, args *pb.StringMessage) (*pb.BooleanMessage, error) {
	err := m.db.RemoveImage(args.Name)
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

	list := m.db.ListAllImages()
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
	slave := NewSlave(m.db.NewSlaveId(), args.Name, args.Channels, args.Architectures)
	m.Slaves = append(m.Slaves, slave)
	logging.Infof("Subscribed slave \"%s\" with id %d\n", slave.Name, slave.Id)
	return slave, nil
}

// Append a job to the list.
func (m *RpcService) appendJob(r *Job) {
	m.jMutex.Lock()
	defer m.jMutex.Unlock()

	// Append job
	m.Jobs = append(m.Jobs, r)

	// Broadcast web clients
	m.master.UpdateStats(func(s *statistics) {
		s.Queued = len(m.Jobs)
	})
}

// Remove a job from the list.
func (m *RpcService) removeJob(j *Job) {
	m.jMutex.Lock()
	defer m.jMutex.Unlock()

	// Find index
	i := -1
	for k, v := range m.Jobs {
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
	m.Jobs, m.Jobs[len(m.Jobs)-1] =
		append(m.Jobs[:i], m.Jobs[i+1:]...), nil

	// Broadcast web clients
	m.master.UpdateStats(func(s *statistics) {
		s.Queued = len(m.Jobs)
	})
}

// Enqueue a job.
func (m *RpcService) enqueueJob(target, arch string, t pb.EnumTargetType) (*Job, error) {
	// Verify if the target exists
	switch t {
	case pb.EnumTargetType_PACKAGE:
		if !m.db.HasPackage(target) {
			return nil, fmt.Errorf("%s package not found", target)
		}
		break
	case pb.EnumTargetType_IMAGE:
		if !m.db.HasImage(target) {
			return nil, fmt.Errorf("%s image not found", target)
		}
		break
	default:
		return nil, fmt.Errorf("Wrong target type specified for \"%s\" (%s)\n", target, arch)
	}

	// Create a job
	j := &Job{
		&api.Job{Id: m.db.NewJobId(),
			Target:       target,
			Architecture: arch,
			Started:      time.Time{},
			Finished:     time.Time{},
			Status:       api.JOB_STATUS_JUST_CREATED,
		},
		jobTargetMap[t],
		make(chan bool),
	}

	// Save on the database
	dbJob := &database.Job{j.Id, j.Target, j.Architecture, j.Started, j.Finished, j.Status}
	err := m.db.SaveJob(dbJob)
	dbJob = nil
	if err != nil {
		panic(err)
	}
	m.appendJob(j)

	// Push it onto the queue
	m.master.buildJobQueue <- j
	logging.Infof("Queued job #%d (target \"%s\" for %s)\n", j.Id, target, arch)
	return j, nil
}

// Calculate statistics of completed and failed jobs.
func (m *RpcService) calculateStats() {
	m.master.UpdateStats(func(s *statistics) {
		s.Completed = 0
		s.Failed = 0

		m.db.FilterJobs(func(job *database.Job) bool {
			if !job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				return false
			}
			if job.Status == api.JOB_STATUS_SUCCESSFUL {
				s.Completed++
			} else if job.Status > api.JOB_STATUS_SUCCESSFUL {
				s.Failed++
			}
			return false
		})
	})
}
