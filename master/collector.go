/****************************************************************************
 * This file is part of Hawaii.
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
	"github.com/hawaii-desktop/builder/common/logging"
	"net/http"
	"sync/atomic"
	"time"
)

// Buffered channel that we can send jobs on
var BuildJobQueue = make(chan *Job, Config.Build.MaxJobs)

func Collector(m *Master, w http.ResponseWriter, r *http.Request) {
	// This is only allowed with a POST
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Package name
	target := r.FormValue("target")
	if target == "" {
		http.Error(w, "You must specify a target", http.StatusBadRequest)
		return
	}

	// Allocate a new global id
	id := atomic.AddUint64(&globalJobId, 1)

	// Create a job
	j := &Job{
		Id:       id,
		Target:   target,
		Started:  time.Time{},
		Finished: time.Time{},
		Status:   JOB_STATUS_JUST_CREATED,
		Channel:  make(chan bool),
	}
	m.appendJob(j)

	// Push it onto the queue
	BuildJobQueue <- j
	logging.Infof("Queued job #%d (target \"%s\")\n", id, target)

	// Reply
	w.WriteHeader(http.StatusCreated)
}
