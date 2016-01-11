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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/database"
	"github.com/hawaii-desktop/builder/logging"
	pb "github.com/hawaii-desktop/builder/protocol"
	"github.com/hawaii-desktop/builder/utils"
	"golang.org/x/net/context"
	"io"
	"os"
	"path/filepath"
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

// Subscribe to the master.
func (m *RpcService) Subscribe(ctx context.Context, args *pb.SubscribeRequest) (*pb.SubscribeResponse, error) {
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

	// Reply
	response := &pb.SubscribeResponse{
		Id:        slave.Id,
		ImagesDir: Config.Storage.ImagesDir,
		RepoUrl:   fmt.Sprintf("%s/packages/fedora/releases/$releasever/$basearch/os", m.master.repoBaseUrl),
	}
	return response, nil
}

// Slave call this to initiate a dialog.
func (m *RpcService) PickJob(stream pb.Builder_PickJobServer) error {
	var slave *Slave = nil

	// Messages that will be streamed to the slave are sent here
	outChannel := make(chan *pb.JobRequest)
	defer func() {
		// Quit the goroutines that are picking up from this channel
		outChannel <- nil
	}()

	for {
		// Read request from the stream
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Slave start
		slaveStart := in.GetSlaveStart()
		if slaveStart != nil {
			// Find the slave
			m.sMutex.Lock()
			for _, s := range m.Slaves {
				if s.Id == slaveStart.Id {
					slave = s
					break
				}
			}
			m.sMutex.Unlock()

			// Can't continue if the slave was not found
			if slave == nil {
				return ErrSlaveNotFound
			}

			// Stream job requests to the slave when the dispatch
			// function send them to the channel we created above
			go func() {
				for {
					select {
					case r := <-outChannel:
						// Quit this go routine when a nil request has been received
						if r == nil {
							return
						}

						// Otherwise stream the request to the slave
						stream.Send(r)
						logging.Infof("Job #%d scheduled on \"%s\"\n", r.Id, slave.Name)
					}
				}
			}()

			// Dispatch jobs to this slave
			m.master.dispatchSlave(slave, outChannel)
		}

		// Job update
		jobUpdate := in.GetJobUpdate()
		if jobUpdate != nil {
			// We need the slave
			if slave == nil {
				return ErrSlaveNotFound
			}

			var job *Job = nil
			m.master.forEachJob(func(curJob *Job) {
				// Skip other jobs
				if curJob.Id == jobUpdate.Id {
					job = curJob
				}
			})
			if job == nil {
				logging.Errorf("Cannot find job #%d\n", jobUpdate.Id)
				return ErrJobNotFound
			}

			// Update the status
			job.Status = jobStatusMap[jobUpdate.Status]

			// Handle status change
			if job.Status >= builder.JOB_STATUS_SUCCESSFUL && job.Status <= builder.JOB_STATUS_CRASHED {
				// Update finished time and notify
				job.Finished = time.Now()

				// Log the status
				if job.Status == builder.JOB_STATUS_SUCCESSFUL {
					logging.Infof("Job #%d completed successfully on \"%s\"\n",
						job.Id, slave.Name)

					// Update repodata and repoview
					m.master.repoDataQueue <- true
				} else {
					logging.Errorf("Job #%d failed on \"%s\"\n",
						job.Id, slave.Name)
				}

				// Send status notification(s)
				m.master.sendStatusNotifications(job)

				// Remove from the list
				m.master.removeJob(job)

				// Proceed to the next job
				job.Channel <- true
			} else {
				logging.Tracef("Change job #%d status to \"%s\"\n",
					jobUpdate.Id, builder.JobStatusDescriptionMap[job.Status])
			}

			// Save on the database
			m.master.saveDatabaseJob(job)

			// Update Web socket clients
			m.master.updateStatistics()
			m.master.updateAllJobs()
		}

		// Build step update
		stepUpdate := in.GetStepUpdate()
		if stepUpdate != nil {
			// We need the slave
			if slave == nil {
				return ErrSlaveNotFound
			}

			// Do we have a valid job here?
			var job *Job = nil
			m.master.forEachJob(func(curJob *Job) {
				// Skip other jobs
				if curJob.Id == stepUpdate.JobId {
					job = curJob
				}
			})
			if job == nil {
				logging.Errorf("Cannot find job #%d\n", stepUpdate.JobId)
				return ErrJobNotFound
			}

			// Append or replace the build step
			job.Mutex.Lock()
			found := false
			for _, step := range job.Steps {
				if step.Name == stepUpdate.Name {
					step.Started = time.Unix(0, stepUpdate.Started)
					step.Finished = time.Unix(0, stepUpdate.Finished)
					step.Summary = utils.MapSliceString(stepUpdate.Summary)
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
					Logs:     stepUpdate.Logs,
				}
				job.Steps = append(job.Steps, step)
			}
			job.Mutex.Unlock()

			// Save on the database
			m.master.saveDatabaseJob(job)

			// Update Web socket clients
			m.master.updateStatistics()
			m.master.updateAllJobs()
		}
	}

	return nil
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

// Upload a file from slave to master.
func (m *RpcService) Upload(stream pb.Builder_UploadServer) error {
	var file *os.File = nil

	// Count how many bytes we write
	total := int64(0)

	// SHA256 hash
	hasher := sha256.New()

	// Regular expressions for RPMs
	re := regexp.MustCompile(`^(.+)\.([a-z0-9\-_]+)\.rpm$`)

	for {
		// Read request from the stream
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Create the file if needed
		request := in.GetRequest()
		if request != nil {
			// Determine the final location
			destpath := ""
			m := re.FindStringSubmatch(request.FileName)
			if len(m) == 3 {
				letter := m[1][:1]
				destpath = fmt.Sprintf("%s/fedora/releases/%s/%s/os/Packages/%s/%s",
					Config.Storage.RepositoryDir, request.ReleaseVer, request.BaseArch,
					letter, request.FileName)
			} else {
				return stream.SendAndClose(&pb.UploadResponse{total, "invalid file name"})
			}

			// Create all directories
			logging.Infof("Receiving upload of \"%s\"...\n", request.FileName)
			err = os.MkdirAll(filepath.Dir(destpath), 0755)
			if err != nil {
				return stream.SendAndClose(&pb.UploadResponse{total, err.Error()})
			}

			// Create the file
			file, err = os.Create(destpath)
			if err != nil {
				return stream.SendAndClose(&pb.UploadResponse{total, err.Error()})
			}
		}

		// Write chunks
		chunk := in.GetChunk()
		if chunk != nil {
			// Write chunk
			size, err := file.Write(chunk.Data)
			if err != nil {
				file.Sync()
				file.Close()
				return stream.SendAndClose(&pb.UploadResponse{total, err.Error()})
			}
			total += int64(size)

			// Update hash with this chunk
			hasher.Sum(chunk.Data)
		}
		file.Sync()

		// End transfer
		end := in.GetEnd()
		if end != nil {
			// Close file
			file.Close()

			// Change permission
			file.Chmod(os.FileMode(end.Permission))

			// Verify hash
			hash := hasher.Sum(nil)
			if !bytes.Equal(hash, end.Hash) {
				errMsg := fmt.Errorf("wrong SHA256 hash \"%s\", expected \"%s\"",
					hex.EncodeToString(hash), hex.EncodeToString(end.Hash))
				return stream.SendAndClose(&pb.UploadResponse{total, errMsg.Error()})
			}

			break
		}
	}

	return stream.SendAndClose(&pb.UploadResponse{total, ""})
}

// Download a file from master to slave.
func (m *RpcService) Download(request *pb.DownloadRequest, stream pb.Builder_DownloadServer) error {
	// SHA256 hash
	hasher := sha256.New()

	// Open the file
	file, err := os.Open(request.FileName)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := os.Stat(request.FileName)
	if err != nil {
		return err
	}

	for {
		// Read a chunk and transfer
		chunk := make([]byte, 1024*1024)
		size, err := file.Read(chunk)
		if err == io.EOF {
			break
		}
		if err == nil {
			response := &pb.DownloadResponse{
				Payload: &pb.DownloadResponse_Chunk{
					Chunk: &pb.DownloadChunk{
						Data: chunk[:size],
						Hash: hasher.Sum(chunk[:size]),
					},
				},
			}
			if err := stream.Send(response); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// End transfer
	response := &pb.DownloadResponse{
		Payload: &pb.DownloadResponse_End{
			End: &pb.DownloadEnd{
				Hash: hasher.Sum(nil),
				Size: stat.Size(),
			},
		},
	}
	if err := stream.Send(response); err != nil {
		return err
	}

	return nil
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
