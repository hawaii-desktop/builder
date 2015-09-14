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
	"github.com/hawaii-desktop/builder/common/logging"
	pb "github.com/hawaii-desktop/builder/common/protocol"
	"golang.org/x/net/context"
	"io"
	"sync"
	"time"
)

// Master type exposed via RPC
type Master struct {
	// List of slaves.
	Slaves []*Slave
	// List of jobs to be processed.
	Jobs []*Job
	// Protects slave list.
	sMutex sync.Mutex
	// Protects jobs list.
	jMutex sync.Mutex
}

// Errors
var (
	ErrAlreadySubscribed = errors.New("slave has already subscribed")
	ErrSlaveNotFound     = errors.New("slave not found")
	ErrInvalidSlave      = errors.New("slave is not valid")
	ErrJobNotFound       = errors.New("job not found with that id")
)

// Map to decode job status.
var jobStatusMap = map[pb.EnumJobStatus]JobStatus{
	pb.EnumJobStatus_JOB_STATUS_JUST_CREATED: JOB_STATUS_JUST_CREATED,
	pb.EnumJobStatus_JOB_STATUS_WAITING:      JOB_STATUS_WAITING,
	pb.EnumJobStatus_JOB_STATUS_PROCESSING:   JOB_STATUS_PROCESSING,
	pb.EnumJobStatus_JOB_STATUS_SUCCESSFUL:   JOB_STATUS_SUCCESSFUL,
	pb.EnumJobStatus_JOB_STATUS_FAILED:       JOB_STATUS_FAILED,
	pb.EnumJobStatus_JOB_STATUS_CRASHED:      JOB_STATUS_CRASHED,
}

// Allocate a new Master with an empty list of slaves.
// The slaves list is initially empty and has a capacity as big as the maximum
// number of slaves from the configuration.
// The jobs list is initially empty and has a capacity as big as
// the maximum number of jobs from the configuration.
func NewMaster() *Master {
	m := &Master{}
	m.Slaves = make([]*Slave, 0, Config.Build.MaxSlaves)
	m.Jobs = make([]*Job, 0, Config.Build.MaxJobs)
	return m
}

// Slave call this to initiate a dialog.
func (m *Master) Subscribe(stream pb.Builder_SubscribeServer) error {
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
		response := &pb.JobDispatchRequest{
			Id:     j.Id,
			Target: j.Target,
		}
		reply := &pb.OutputMessage{
			Payload: &pb.OutputMessage_JobDispatch{
				JobDispatch: response,
			},
		}
		stream.Send(reply)
		logging.Infof("Job #%d scheduled on \"%s\"\n", j.Id, s.Name)

	}

	// Slave loop
	var slaveLoop = func(s *Slave) {
		for {
			// Do not queue a slave that suddenly unregisters itself
			if !s.Subscribed || !s.Active {
				return
			}

			// Add to the queue
			s.QueueChannel <- s.JobChannel

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
			if j.Status >= JOB_STATUS_SUCCESSFUL && j.Status <= JOB_STATUS_CRASHED {
				// Update finished time and notify
				j.Finished = time.Now()

				// Log the status
				if j.Status == JOB_STATUS_SUCCESSFUL {
					logging.Infof("Job #%d completed successfully on \"%s\"\n",
						j.Id, slave.Name)
				} else {
					logging.Errorf("Job #%d failed on \"%s\"\n",
						j.Id, slave.Name)
				}

				// TODO: Save on the database

				// Remove from the list
				m.removeJob(j)

				// Proceed to the next job
				j.Channel <- true
			} else {
				logging.Tracef("Change job #%d status to \"%s\"\n",
					jobUpdate.Id, jobStatusDescriptionMap[j.Status])
			}
		}
	}
}

// Unregister a slave and stop it immediately.
// Jobs will not be dispatched to this slave until it has subscribed again.
func (m *Master) Unsubscribe(ctx context.Context, args *pb.UnsubscribeRequest) (*pb.UnsubscribeResponse, error) {
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

// Create a slave from the subscription request.
func (m *Master) createSlave(args *pb.SubscribeRequest) (*Slave, error) {
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
	slave := NewSlave(args.Name, args.Channels, args.Architectures)
	m.Slaves = append(m.Slaves, slave)
	logging.Infof("Subscribed slave \"%s\" with id %d\n", slave.Name, slave.Id)
	return slave, nil
}

// Append a job to the list.
func (m *Master) appendJob(r *Job) {
	m.jMutex.Lock()
	m.Jobs = append(m.Jobs, r)
	m.jMutex.Unlock()
}

// Remove a job from the list.
func (m *Master) removeJob(j *Job) {
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
}
