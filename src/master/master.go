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
	"encoding/json"
	"github.com/hawaii-desktop/builder/src/api"
	"github.com/hawaii-desktop/builder/src/database"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/hawaii-desktop/builder/src/webserver"
	"sync"
	"time"
)

// Master.
type Master struct {
	// Database.
	db *database.Database
	// Web socket hub.
	hub *webserver.WebSocketHub
	// Buffered channel that we can send jobs on.
	buildJobQueue chan *Job
	// Buffered channel that holds the job channels from each slave.
	slaveQueue chan chan *Job
	// Broadcast queue for the web socket.
	webSocketQueue chan interface{}
	// Current statistics.
	stats statistics
	// Mutext that protects statistics.
	sMutex sync.Mutex
}

// Generic request received from the Web user interface.
type request struct {
	Type int `json:"type"`
}

// Generic message sent to the Web user interface.
type message struct {
	Type int         `json:"type"`
	Data interface{} `json:"data,omitifempty"`
}

// Message types
const (
	WEB_SOCKET_STATISTICS = iota
	WEB_SOCKET_QUEUED_JOBS
	WEB_SOCKET_DISPATCHED_JOBS
	WEB_SOCKET_COMPLETED_JOBS
	WEB_SOCKET_FAILED_JOBS
)

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

// Jobs list to show on the Web user interface.
type jobsList struct {
	Id           uint64    `json:"id"`
	Target       string    `json:"target"`
	Architecture string    `json:"arch"`
	Started      time.Time `json:"started"`
	Finished     time.Time `json:"finished"`
}

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
		buildJobQueue:  make(chan *Job, Config.Build.MaxJobs),
		slaveQueue:     make(chan chan *Job, Config.Build.MaxSlaves),
		webSocketQueue: make(chan interface{}),
		stats:          statistics{0, 0, 0, 0, 0, 0},
	}, nil
}

// Close the database.
func (m *Master) Close() {
	m.db.Close()
	m.db = nil
}

// Update statistics and send an event through the web socket.
func (m *Master) UpdateStats(f statisticsUpdateFunc) {
	m.sMutex.Lock()
	defer m.sMutex.Unlock()

	f(&m.stats)

	m.SendStats()

	m.sendJobsListToWebSocket(WEB_SOCKET_QUEUED_JOBS)
	m.sendJobsListToWebSocket(WEB_SOCKET_DISPATCHED_JOBS)
}

// Send statistics through the web socket.
func (m *Master) SendStats() {
	m.webSocketQueue <- &message{Type: WEB_SOCKET_STATISTICS, Data: m.stats}
}

// Start processing
func (m *Master) Process() {
	// Dispatch jobs
	go func() {
		for {
			select {
			case j := <-m.buildJobQueue:
				logging.Tracef("About to dispatch job #%d...\n", j.Id)
				go func() {
					// Update job
					j.Started = time.Now()
					j.Status = api.JOB_STATUS_WAITING

					// Dispatch
					slave := <-m.slaveQueue
					logging.Tracef("Dispatching job #%d...\n", j.Id)
					slave <- j

					// Update statistics
					m.UpdateStats(func(s *statistics) {
						s.Queued--
						s.Dispatched++
					})
				}()
			}
		}
	}()

	// Queue events to the web socket
	go func() {
		for {
			select {
			case e := <-m.webSocketQueue:
				if e != nil {
					m.hub.Broadcast(e)
				}
			}
		}
	}()
}

// Process a request coming from the Web socket.
func (m *Master) processWebSocketRequest(msg []byte) error {
	var r *request
	err := json.Unmarshal(msg, &r)
	if err != nil {
		return err
	}

	if r.Type == WEB_SOCKET_STATISTICS {
		return nil
	}

	m.sendJobsListToWebSocket(r.Type)
	return nil
}

// Send the jobs list to the Web socket.
func (m *Master) sendJobsListToWebSocket(reqType int) {
	var data []*jobsList
	m.db.FilterJobs(func(job *database.Job) bool {
		// Completed and failed jobs are interesting only if finished in the last 48 hours
		if reqType == WEB_SOCKET_COMPLETED_JOBS || reqType == WEB_SOCKET_FAILED_JOBS {
			if !job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				return false
			}
		}

		// Append jobs depending on the request type
		if (reqType == WEB_SOCKET_QUEUED_JOBS && job.Status >= api.JOB_STATUS_JUST_CREATED && job.Status <= api.JOB_STATUS_WAITING) ||
			(reqType == WEB_SOCKET_DISPATCHED_JOBS && job.Status == api.JOB_STATUS_PROCESSING) ||
			(reqType == WEB_SOCKET_COMPLETED_JOBS && job.Status == api.JOB_STATUS_SUCCESSFUL) ||
			(reqType == WEB_SOCKET_FAILED_JOBS && job.Status >= api.JOB_STATUS_FAILED) {
			data = append(data, &jobsList{job.Id, job.Target, job.Architecture, job.Started, job.Finished})
		}

		return false
	})
	m.webSocketQueue <- &message{Type: reqType, Data: data}
}
