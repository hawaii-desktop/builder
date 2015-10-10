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
	// Send statistics as soon as a client connects
	m.updateStatistics()

	// Receive messages from the Web UI
	go func() {
		for {
			// TODO: Find a way to quit this goroutine
			select {
			case msg := <-c.Outgoing:
				var r *wsRequest
				err := json.Unmarshal(msg, &r)
				if err != nil {
					logging.Errorf("Error processing request from Web socket: %s\n", err)
					return
				}

				if r.Type > WEB_SOCKET_STATISTICS && r.Type < WEB_SOCKET_JOB {
					m.updateJobs(r.Type)
				}

				if r.Type == WEB_SOCKET_JOB {
					m.updateJob(r.Id)
				}
				break
			}
		}
	}()
}

// Handle Web socket connection unregistration.
func (m *Master) WebSocketConnectionUnregistration(c *webserver.WebSocketConnection) {
}

// Update statistics.
func (m *Master) updateStatistics() {
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
			break
		case builder.JOB_STATUS_WAITING:
			m.stats.Queued++
			break
		case builder.JOB_STATUS_PROCESSING:
			m.stats.Dispatched++
			break
		case builder.JOB_STATUS_SUCCESSFUL:
			if job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				m.stats.Completed++
			}
			break
		case builder.JOB_STATUS_FAILED:
			if job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				m.stats.Failed++
			}
			break
		case builder.JOB_STATUS_CRASHED:
			if job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				m.stats.Failed++
			}
			break
		}
	})
	m.webSocketQueue <- &wsResponse{Type: WEB_SOCKET_STATISTICS, Data: m.stats}
}

// Send a specific job to the Web socket.
func (m *Master) updateJob(id uint64) {
	job := m.db.GetJob(id)
	if job != nil {
		m.webSocketQueue <- &wsResponse{Type: WEB_SOCKET_JOB, Data: job}
	}
}

// Send the jobs list to the Web socket.
func (m *Master) updateJobs(reqType int) {
	jobs := m.db.FilterJobs(func(job *builder.Job) bool {
		// Completed and failed jobs are interesting only if finished in the last 48 hours
		if reqType == WEB_SOCKET_COMPLETED_JOBS || reqType == WEB_SOCKET_FAILED_JOBS {
			if !job.Finished.After(time.Now().Add(-48 * time.Hour)) {
				return false
			}
		}

		// Append jobs depending on the request type
		if (reqType == WEB_SOCKET_QUEUED_JOBS && job.Status >= builder.JOB_STATUS_JUST_CREATED && job.Status <= builder.JOB_STATUS_WAITING) ||
			(reqType == WEB_SOCKET_DISPATCHED_JOBS && job.Status == builder.JOB_STATUS_PROCESSING) ||
			(reqType == WEB_SOCKET_COMPLETED_JOBS && job.Status == builder.JOB_STATUS_SUCCESSFUL) ||
			(reqType == WEB_SOCKET_FAILED_JOBS && job.Status >= builder.JOB_STATUS_FAILED) {
			return true
		}

		return false
	})
	m.webSocketQueue <- &wsResponse{Type: reqType, Data: jobs}
}

// Send the jobs list to the Web socket regardless of the status.
func (m *Master) updateAllJobs() {
	m.updateJobs(WEB_SOCKET_QUEUED_JOBS)
	m.updateJobs(WEB_SOCKET_DISPATCHED_JOBS)
	m.updateJobs(WEB_SOCKET_COMPLETED_JOBS)
	m.updateJobs(WEB_SOCKET_FAILED_JOBS)
}
