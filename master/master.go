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
	"fmt"
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/database"
	"github.com/hawaii-desktop/builder/logging"
	"github.com/hawaii-desktop/builder/webserver"
	"net"
	"os"
	"sync"
	"time"
)

// Master.
type Master struct {
	// Database.
	db *database.Database
	// Web socket hub.
	hub *webserver.WebSocketHub
	// Web socket client subscriptions.
	subscriptions map[*webserver.WebSocketConnection]*wsSubscription
	// Buffered channel that we can send jobs on.
	buildJobQueue chan *Job
	// Map a slave topic (that is a combination of what job types and
	// architectures supported by a slave, for example package/x86_64 for
	// x86_64 packages) to a buffered channel that holds the job channels
	// from each slave.
	slaveQueues map[string]chan chan *Job
	// Broadcast queue for the web socket.
	webSocketQueue chan interface{}
	// List of jobs to be processed.
	jobs []*Job
	// Protects jobs list.
	jobsMutex sync.Mutex
	// Current statistics.
	stats statistics
	// Mutext that protects statistics.
	sMutex sync.Mutex
	// Repository base URL.
	repoBaseUrl string
}

// Statistics to show on the Web user interface.
type statistics struct {
	Queued     int `json:"queued"`
	Dispatched int `json:"dispatched"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Packages   int `json:"packages"`
	Images     int `json:"images"`
}

// Update function.
type statisticsUpdateFunc func(s *statistics)

// Create a new master.
// This also create or open the database.
func NewMaster(hub *webserver.WebSocketHub) (*Master, error) {
	db, err := database.NewDatabase(Config.Server.Database)
	if err != nil {
		return nil, err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", Config.Server.HttpAddress)
	if err != nil {
		return nil, err
	}

	addr := tcpAddr.String()
	if tcpAddr.IP == nil {
		addr = "localhost" + addr
	}

	return &Master{
		db:             db,
		hub:            hub,
		subscriptions:  make(map[*webserver.WebSocketConnection]*wsSubscription),
		buildJobQueue:  make(chan *Job, Config.Build.MaxJobs),
		slaveQueues:    make(map[string]chan chan *Job),
		webSocketQueue: make(chan interface{}),
		jobs:           make([]*Job, 0, Config.Build.MaxJobs),
		stats:          statistics{0, 0, 0, 0, 0, 0},
		repoBaseUrl:    "http://" + addr + "/repo",
	}, nil
}

// Close the database.
func (m *Master) Close() {
	m.db.Close()
	m.db = nil
}

// Prepare the topics.
func (m *Master) PrepareTopics() {
	// Create a bufferer channel of jobs for the topic, unless
	// it has already been created
	for _, ttype := range []string{"package", "image"} {
		for _, arch := range m.db.ListArchitectures() {
			topic := ttype + "/" + arch
			if _, ok := m.slaveQueues[topic]; !ok {
				m.slaveQueues[topic] = make(chan chan *Job, Config.Build.MaxSlaves)
			}
		}
	}
}

// Create storage directories.
func (m *Master) CreateStorage() error {
	if err := os.MkdirAll(Config.Storage.RepositoryDir, 0755); err != nil {
		fmt.Errorf("Failed to create main repository directory \"%s\": %s\n", Config.Storage.RepositoryDir, err)
	}
	if err := os.MkdirAll(Config.Storage.ImagesDir, 0755); err != nil {
		fmt.Errorf("Failed to create images storage \"%s\": %s\n", Config.Storage.ImagesDir, err)
	}
	return nil
}

// Dispatch jobs.
func (m *Master) Dispatch() {
	for {
		select {
		case j := <-m.buildJobQueue:
			logging.Tracef("About to dispatch job #%d...\n", j.Id)
			go func() {
				// Update job
				j.Started = time.Now()
				j.Status = builder.JOB_STATUS_WAITING

				// Save on the database
				m.saveDatabaseJob(j)

				// Update Web socket clients
				m.updateStatistics()
				m.updateAllJobs()

				// Dispatch job based on job type and architecture
				topic := j.TopicName()
				slaveQueue := <-m.slaveQueues[topic]
				logging.Tracef("Dispatching job #%d (topic %s)...\n", j.Id, topic)
				slaveQueue <- j
			}()
		}
	}
}

// Queue events to the Web socket.
func (m *Master) DeliverWebSocketEvents() {
	for {
		select {
		case e := <-m.webSocketQueue:
			if e != nil {
				m.hub.Broadcast(e)
			}
		}
	}
}
