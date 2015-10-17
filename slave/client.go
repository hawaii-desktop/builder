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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/logging"
	pb "github.com/hawaii-desktop/builder/protocol"
	"github.com/hawaii-desktop/builder/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Store important data used during the life time of the slave.
type Client struct {
	// RPC proxy.
	client pb.BuilderClient
	// Identifier for this slave, attributed after subscription.
	// Its value is 0 when unsubscribed.
	slaveId uint64
	// Main repository path on master.
	mainRepoDir string
	// Staging repository path on master.
	stagingRepoDir string
	// Images repository path on master
	imagesDir string
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
					Summary:  utils.MapStringSlice(bs.summary),
					Logs:     bs.logs,
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
				c.mainRepoDir = subscription.MainRepoDir
				c.stagingRepoDir = subscription.StagingRepoDir
				c.imagesDir = subscription.ImagesDir
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
						case <-j.artifactsChannel:
							if err := c.UploadArtifacts(j.artifacts); err != nil {
								j.Status = builder.JOB_STATUS_FAILED
								j.Finished = time.Now()
								logging.Errorln(err)
							}
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
				Types:         strings.Split(Config.Slave.Types, ","),
				Architectures: strings.Split(Config.Slave.Architectures, ","),
			},
		},
	}
	stream.Send(args)

	// Wait until the stream is clsed
	go func() {
		<-wait
		stream.CloseSend()
	}()

	return nil
}

// Unsubscribe from the master.
func (c *Client) Unsubscribe() error {
	args := &pb.UnsubscribeRequest{Id: c.slaveId}
	reply, err := c.client.Unsubscribe(context.Background(), args)
	if reply.Result {
		c.slaveId = 0
		c.mainRepoDir = ""
		c.stagingRepoDir = ""
		c.imagesDir = ""
	}
	return err
}

// Upload an artifact to the master.
func (c *Client) UploadArtifact(artifact *Artifact) error {
	// Logging
	logging.Infof("Uploading \"%s\" to the staging repository...\n",
		filepath.Base(artifact.Source))

	// Determine how many bytes are left to send
	stat, err := os.Stat(artifact.Source)
	if err != nil {
		return fmt.Errorf("Failed to stat \"%s\": %s", err)
	}

	// Open the file
	file, err := os.Open(artifact.Source)
	if err != nil {
		return fmt.Errorf("Failed to open \"%s\": %s", err)
	}

	// SHA256 hash
	hasher := sha256.New()

	// Open the stream
	stream, err := c.client.Upload(context.Background())
	if err != nil {
		return err
	}

	// Begin transfer
	args := &pb.UploadMessage{
		Payload: &pb.UploadMessage_Request{
			Request: &pb.UploadRequest{
				FileName: artifact.Destination,
			},
		},
	}
	if err := stream.Send(args); err != nil {
		stream.CloseSend()
		return fmt.Errorf("Failed to start upload of \"%s\": %s",
			filepath.Base(artifact.Source), err)
	}

	// Read a chunk and transfer
	file.Seek(0, 0)
	for {
		// Send 1MB chunks
		chunk := make([]byte, 1024*1024)
		size, err := file.Read(chunk)
		if err == nil {
			args := &pb.UploadMessage{
				Payload: &pb.UploadMessage_Chunk{
					Chunk: &pb.UploadChunk{
						Data: chunk,
						Hash: hasher.Sum(chunk[:size]),
					},
				},
			}
			if err := stream.Send(args); err != nil {
				stream.CloseSend()
				return fmt.Errorf("Unable to upload a chunk of \"%s\": %s",
					filepath.Base(artifact.Source), err)
			}
		} else if err == io.EOF {
			break
		} else {
			stream.CloseSend()
			return fmt.Errorf("Failed to read \"%s\": %s",
				filepath.Base(artifact.Source), err)
		}
	}

	// Close file
	file.Close()

	// End transfer
	args = &pb.UploadMessage{
		Payload: &pb.UploadMessage_End{
			End: &pb.UploadEnd{
				Hash:       hasher.Sum(nil),
				Permission: artifact.Permission,
			},
		},
	}
	if err := stream.Send(args); err != nil {
		stream.CloseSend()
		return fmt.Errorf("Failed to end upload of \"%s\": %s",
			filepath.Base(artifact.Source), err)
	}

	// Close stream and receive reply
	reply, err := stream.CloseAndRecv()
	if err == nil {
		if reply.TotalSize != stat.Size() {
			return fmt.Errorf("Upload of \"%s\" failed: uploaded %d bytes but file is %d bytes",
				filepath.Base(artifact.Source), reply.TotalSize, stat.Size())
		}
	} else {
		return fmt.Errorf("Error uploading \"%s\": %s",
			filepath.Base(artifact.Source), err)
	}

	return nil
}

// Upload artifacts to the master.
func (c *Client) UploadArtifacts(artifacts []*Artifact) error {
	var wg sync.WaitGroup
	var mutex sync.RWMutex
	var errors []string

	for _, artifact := range artifacts {
		wg.Add(1)
		go func(artifact *Artifact) {
			defer wg.Done()
			if err := c.UploadArtifact(artifact); err != nil {
				mutex.Lock()
				errors = append(errors, err.Error())
				mutex.Unlock()
			}
		}(artifact)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Failed to upload artifacts:\n%s\n", strings.Join(errors, "\n"))
	}

	return nil
}

// Download a file from the master.
func (c *Client) DownloadFile(srcfilename, dstfilename string) error {
	// Open file
	file, err := os.Create(dstfilename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Initiate download
	stream, err := c.client.Download(context.Background(), &pb.DownloadRequest{srcfilename})
	if err != nil {
		return err
	}

	// Count total size
	total := int64(0)

	// SHA256 hash
	hasher := sha256.New()

	// Write chunks received from master
	for {
		r, err := stream.Recv()
		if err != nil {
			return err
		}

		chunk := r.GetChunk()
		if chunk != nil {
			// Write chunk
			size, err := file.Write(chunk.Data)
			if err != nil {
				return err
			}
			file.Sync()

			// Size check
			if size != len(chunk.Data) {
				return fmt.Errorf("chunk size mismatch download (%d bytes wrote, %d expected): %s",
					size, len(chunk.Data), err)
			}

			// Hash check
			hash := hasher.Sum(chunk.Data)
			if !bytes.Equal(hash, chunk.Hash) {
				return fmt.Errorf("wrong SHA256 hash \"%s\" for the chunk, expected \"%s\"",
					hex.EncodeToString(hash), hex.EncodeToString(chunk.Hash))
			}

			// Increment size
			total += int64(size)
		}

		end := r.GetEnd()
		if end != nil {
			// Size check
			if total != end.Size {
				return fmt.Errorf("size mismatch: %d bytes received, %d expected",
					total, end.Size)
			}

			// Hash check
			hash := hasher.Sum(nil)
			if !bytes.Equal(hash, end.Hash) {
				return fmt.Errorf("wrong SHA256 hash \"%s\", expected \"%s\"",
					hex.EncodeToString(hash), hex.EncodeToString(end.Hash))
			}
		}
	}

	return nil
}
