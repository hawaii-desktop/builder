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
func (c *Client) Subscribe() (context.Context, error) {
	var ctx context.Context

	request := &pb.SubscribeRequest{
		Name:          Config.Slave.Name,
		Types:         strings.Split(Config.Slave.Types, ","),
		Architectures: strings.Split(Config.Slave.Architectures, ","),
	}
	response, err := c.client.Subscribe(context.Background(), request)
	if err != nil {
		return ctx, err
	}

	data := &SlaveData{
		Id:        response.Id,
		ImagesDir: response.ImagesDir,
		RepoUrl:   response.RepoUrl,
	}
	logging.Infof("Slave subscribed with id %d\n", data.Id)
	ctx = NewContext(context.Background(), data)

	return ctx, nil
}

// PickJobs picks up jobs from the master and send back updates.
func (c *Client) PickJob(ctx context.Context, waitc <-chan struct{}) error {
	// Get data from context
	data, ok := FromContext(ctx)
	if !ok {
		return fmt.Errorf("no data from context")
	}

	// Initiate communication
	stream, err := c.client.PickJob(context.Background())
	if err != nil {
		return err
	}

	// Function that send job updates back to the master
	var sendJobUpdate = func(j *Job) {
		args := &pb.PickJobRequest{
			Payload: &pb.PickJobRequest_JobUpdate{
				JobUpdate: &pb.JobUpdateRequest{
					Id:     j.Id,
					Status: jobStatusMap[j.Status],
				},
			},
		}
		stream.Send(args)
	}

	// Function that send job updates back to the master
	var sendStepUpdate = func(j *Job, bs *BuildStep) {
		args := &pb.PickJobRequest{
			Payload: &pb.PickJobRequest_StepUpdate{
				StepUpdate: &pb.StepUpdateRequest{
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

	// Start the dispatcher
	args := &pb.PickJobRequest{
		Payload: &pb.PickJobRequest_SlaveStart{
			SlaveStart: &pb.SlaveStartRequest{
				Id: data.Id,
			},
		},
	}
	stream.Send(args)

	// Read from the stream
	for {
		// Receive
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Read build information from the request
		var (
			target string
			arch   string
		)
		pkg := in.GetPackage()
		if pkg != nil {
			target = pkg.Name
			arch = pkg.Architectures[0]
		}
		img := in.GetImage()
		if img != nil {
			target = img.Name
			arch = img.Architectures[0]
		}

		// Create a new job
		logging.Infof("Processing job #%d (target \"%s\" for %s)\n",
			in.Id, target, arch)
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
		j := NewJob(ctx, in.Id, target, arch, &TargetInfo{pkgInfo, imgInfo})

		// Send updates back to master
		go func(j *Job) {
			for {
				select {
				case <-j.UpdateChannel:
					sendJobUpdate(j)
				case bs := <-j.stepUpdateQueue:
					sendStepUpdate(j, bs)
				case <-j.artifactsChannel:
					if err := c.UploadArtifacts(j.artifacts); err != nil {
						j.Status = builder.JOB_STATUS_FAILED
						j.Finished = time.Now()
						logging.Errorln(err)
					}
				case <-j.CloseChannel:
					j = nil
					return
				case <-waitc:
					return
				}
			}
		}(j)

		// Process
		c.jobQueue <- j
	}

	// Close the stream
	go func() {
		<-waitc
		stream.CloseSend()
	}()

	return nil
}

// Unsubscribe from the master.
func (c *Client) Unsubscribe(ctx context.Context) error {
	data, ok := FromContext(ctx)
	if !ok {
		return fmt.Errorf("no data from context")
	}

	args := &pb.UnsubscribeRequest{Id: data.Id}
	_, err := c.client.Unsubscribe(ctx, args)
	return err
}

// Upload an artifact to the master.
func (c *Client) UploadArtifact(artifact *Artifact) error {
	// Logging
	logging.Infof("Uploading \"%s\" to the staging repository...\n",
		filepath.Base(artifact.FileName))

	// Determine how many bytes are left to send
	stat, err := os.Stat(artifact.FileName)
	if err != nil {
		return fmt.Errorf("Failed to stat \"%s\": %s", err)
	}

	// Open the file
	file, err := os.Open(artifact.FileName)
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
				FileName:   filepath.Base(artifact.FileName),
				ReleaseVer: artifact.ReleaseVer,
				BaseArch:   artifact.BaseArch,
			},
		},
	}
	if err := stream.Send(args); err != nil {
		return fmt.Errorf("Failed to start upload of \"%s\": %s",
			filepath.Base(artifact.FileName), err)
	}

	// Read a chunk and transfer
	file.Seek(0, 0)
	for {
		// Send 1MB chunks
		chunk := make([]byte, 1024*1024)
		size, err := file.Read(chunk)
		if err == nil {
			// Update hash with this chunk
			hasher.Sum(chunk[:size])

			// Send chunk
			args := &pb.UploadMessage{
				Payload: &pb.UploadMessage_Chunk{
					Chunk: &pb.UploadChunk{
						Data: chunk[:size],
					},
				},
			}
			if err := stream.Send(args); err != nil {
				return fmt.Errorf("Unable to upload a chunk of \"%s\": %s",
					filepath.Base(artifact.FileName), err)
			}
		} else if err == io.EOF {
			break
		} else {
			return fmt.Errorf("Failed to read \"%s\": %s",
				filepath.Base(artifact.FileName), err)
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
		return fmt.Errorf("Failed to end upload of \"%s\": %s",
			filepath.Base(artifact.FileName), err)
	}

	// Close stream and receive reply
	reply, err := stream.CloseAndRecv()
	if err == nil {
		if reply.TotalSize != stat.Size() {
			return fmt.Errorf("Upload of \"%s\" failed: uploaded %d bytes but file is %d bytes",
				filepath.Base(artifact.FileName), reply.TotalSize, stat.Size())
		}
	} else {
		return fmt.Errorf("Error uploading \"%s\": %s",
			filepath.Base(artifact.FileName), err)
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
		return fmt.Errorf("Failed to upload artifacts:\n%s", strings.Join(errors, "\n"))
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
