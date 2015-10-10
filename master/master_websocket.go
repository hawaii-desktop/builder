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
	"encoding/json"
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/logging"
	"github.com/hawaii-desktop/builder/webserver"
	"time"
)

// Web socket subscription for certain events.
type wsSubscription struct {
	Type int
	C    chan bool
}

// Generic request received from the Web user interface.
type wsRequest struct {
	Type int    `json:"type"`
	Id   uint64 `json:"id,omitempty"`
}

// Generic message sent to the Web user interface.
type wsResponse struct {
	Type int         `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// Message types.
const (
	WEB_SOCKET_STATISTICS = iota
	WEB_SOCKET_QUEUED_JOBS
	WEB_SOCKET_DISPATCHED_JOBS
	WEB_SOCKET_COMPLETED_JOBS
	WEB_SOCKET_FAILED_JOBS
	WEB_SOCKET_JOB
)

// Handle Web socket connection registration.
func (m *Master) WebSocketConnectionRegistration(c *webserver.WebSocketConnection) {
	// Add the subscription
	m.subscriptions[c] = &wsSubscription{-1, make(chan bool)}

	// Receive messages from the Web UI and quit the goroutine when
	// the client has unregistered
	go func() {
		for {
			select {
			case msg := <-c.Outgoing:
				var r *wsRequest
				err := json.Unmarshal(msg, &r)
				if err != nil {
					logging.Errorf("Error processing request from Web socket: %s\n", err)
					return
				}

				m.subscriptions[c].Type = r.Type
				switch {
				case r.Type == WEB_SOCKET_STATISTICS:
					m.calculateStatistics()
					m.sendStatistics(c)
				case r.Type >= WEB_SOCKET_QUEUED_JOBS && r.Type <= WEB_SOCKET_FAILED_JOBS:
					m.updateJobsForConnection(r.Type, c)
				case r.Type == WEB_SOCKET_JOB:
					m.updateJobForConnection(r.Id, c)
				}
			case <-m.subscriptions[c].C:
				return
			}
		}
	}()
}

// Handle Web socket connection unregistration.
func (m *Master) WebSocketConnectionUnregistration(c *webserver.WebSocketConnection) {
	m.subscriptions[c].C <- true
	close(m.subscriptions[c].C)
	delete(m.subscriptions, c)
}

// Calculate statistics.
func (m *Master) calculateStatistics() {
	// Prevent other goroutines from updating statistics
	m.sMutex.Lock()
	defer m.sMutex.Unlock()

	// Update statistics
	m.stats.Queued = 0
	m.stats.Dispatched = 0
	m.stats.Completed = 0
	m.stats.Failed = 0
	m.db.ForEachJob(func(job *builder.Job) {
		switch job.Status {
		case builder.JOB_STATUS_JUST_CREATED:
			m.stats.Queued++
		case builder.JOB_STATUS_WAITING:
			m.stats.Queued++
		case builder.JOB_STATUS_PROCESSING:
			m.stats.Dispatched++
		case builder.JOB_STATUS_SUCCESSFUL:
			if job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				m.stats.Completed++
			}
		case builder.JOB_STATUS_FAILED:
			if job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				m.stats.Failed++
			}
		case builder.JOB_STATUS_CRASHED:
			if job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				m.stats.Failed++
			}
		}
	})
}

// Send updated statistics to all Web socket connections.
func (m *Master) updateStatistics() {
	// Recalculate the statistics and send
	m.calculateStatistics()
	for c, v := range m.subscriptions {
		if v.Type == WEB_SOCKET_STATISTICS {
			m.sendStatistics(c)
		}
	}
}

// Send updated statistics to a specific Web socket connection.
func (m *Master) sendStatistics(c *webserver.WebSocketConnection) {
	// Recalculate the statistics and send
	err := c.Write(&wsResponse{Type: WEB_SOCKET_STATISTICS, Data: m.stats})
	if err != nil {
		logging.Errorf("Unable to send statistics to the Web socket: %s\n", err)
	}
}

// Broadcast statistics to all Web socket connections.
func (m *Master) broadcastStatistics() {
	// Recalculate the statistics and send
	m.calculateStatistics()
	m.webSocketQueue <- &wsResponse{Type: WEB_SOCKET_STATISTICS, Data: m.stats}
}

// Send a specific job to all Web socket connections.
func (m *Master) updateJob(id uint64) {
	job := m.db.GetJob(id)
	if job != nil {
		for c, v := range m.subscriptions {
			if v.Type == WEB_SOCKET_JOB {
				m.updateJobForConnection(id, c)
			}
		}
	}
}

// Send a specific job to the Web socket connection.
func (m *Master) updateJobForConnection(id uint64, c *webserver.WebSocketConnection) {
	job := m.db.GetJob(id)
	if job == nil {
		logging.Errorf("Job #%d not found\n", id)
	} else {
		err := c.Write(&wsResponse{Type: WEB_SOCKET_JOB, Data: job})
		if err != nil {
			logging.Errorf("Unable to send job updates to the Web socket: %s\n", err)
		}
	}
}

// Prepare a Web socket message with the jobs list.
func (m *Master) prepareJobsList(reqType int) *wsResponse {
	jobs := m.db.FilterJobs(func(job *builder.Job) bool {
		// Completed and failed jobs are interesting only if finished in the last 48 hours
		if reqType == WEB_SOCKET_COMPLETED_JOBS || reqType == WEB_SOCKET_FAILED_JOBS {
			if !job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				return false
			}
		}

		// Append jobs depending on the request type
		if reqType == WEB_SOCKET_QUEUED_JOBS {
			logging.Traceln(job.Status, builder.JOB_STATUS_JUST_CREATED, builder.JOB_STATUS_WAITING)
			return job.Status >= builder.JOB_STATUS_JUST_CREATED && job.Status <= builder.JOB_STATUS_WAITING
		}
		if reqType == WEB_SOCKET_DISPATCHED_JOBS {
			return job.Status == builder.JOB_STATUS_PROCESSING
		}
		if reqType == WEB_SOCKET_COMPLETED_JOBS {
			return job.Status == builder.JOB_STATUS_SUCCESSFUL
		}
		if reqType == WEB_SOCKET_FAILED_JOBS {
			return job.Status >= builder.JOB_STATUS_FAILED
		}

		return false
	})
	return &wsResponse{Type: reqType, Data: jobs}
}

// Send the jobs list to all Web socket connections.
func (m *Master) updateJobs(reqType int) {
	msg := m.prepareJobsList(reqType)

	for c, v := range m.subscriptions {
		if v.Type == reqType {
			err := c.Write(msg)
			if err != nil {
				logging.Errorf("Unable to send jobs list to the Web socket: %s\n", err)
			}
		}
	}
}

// Send the jobs list to a specific Web socket connection.
func (m *Master) updateJobsForConnection(reqType int, c *webserver.WebSocketConnection) {
	err := c.Write(m.prepareJobsList(reqType))
	if err != nil {
		logging.Errorf("Unable to send jobs list to the Web socket: %s\n", err)
	}
}

// Send the jobs list to all Web socket connections.
func (m *Master) updateAllJobs() {
	m.updateJobs(WEB_SOCKET_QUEUED_JOBS)
	m.updateJobs(WEB_SOCKET_DISPATCHED_JOBS)
	m.updateJobs(WEB_SOCKET_COMPLETED_JOBS)
	m.updateJobs(WEB_SOCKET_FAILED_JOBS)
}
