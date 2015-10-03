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
	"github.com/hawaii-desktop/builder/src/api"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/hawaii-desktop/builder/src/webserver"
	"sync"
	"time"
)

// Master.
type Master struct {
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

// Create a new master.
func NewMaster(hub *webserver.WebSocketHub) *Master {
	return &Master{
		hub:            hub,
		buildJobQueue:  make(chan *Job, Config.Build.MaxJobs),
		slaveQueue:     make(chan chan *Job, Config.Build.MaxSlaves),
		webSocketQueue: make(chan interface{}),
		stats:          statistics{0, 0, 0, 0, 0, 0},
	}
}

// Update statistics and send an event through the web socket.
func (m *Master) UpdateStats(f statisticsUpdateFunc) {
	m.sMutex.Lock()
	defer m.sMutex.Unlock()

	f(&m.stats)

	m.SendStats()
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
