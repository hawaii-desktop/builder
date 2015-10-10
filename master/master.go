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
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/database"
	"github.com/hawaii-desktop/builder/logging"
	"github.com/hawaii-desktop/builder/webserver"
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
	// Buffered channel that holds the job channels from each slave.
	slaveQueue chan chan *Job
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
}

// Statistics to show on the Web user interface.
type statistics struct {
	Queued     int `json:"queued"`
	Dispatched int `json:"dispatched"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Staging    int `json:"staging"`
	Main       int `json:"main"`
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

	return &Master{
		db:             db,
		hub:            hub,
		subscriptions:  make(map[*webserver.WebSocketConnection]*wsSubscription),
		buildJobQueue:  make(chan *Job, Config.Build.MaxJobs),
		slaveQueue:     make(chan chan *Job, Config.Build.MaxSlaves),
		webSocketQueue: make(chan interface{}),
		jobs:           make([]*Job, 0, Config.Build.MaxJobs),
		stats:          statistics{0, 0, 0, 0, 0, 0},
	}, nil
}

// Close the database.
func (m *Master) Close() {
	m.db.Close()
	m.db = nil
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

				// Dispatch
				slave := <-m.slaveQueue
				logging.Tracef("Dispatching job #%d...\n", j.Id)
				slave <- j
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
