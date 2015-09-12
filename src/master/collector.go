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

package main

import (
	"../common/logging"
	"net/http"
	"sync/atomic"
	"time"
)

// Buffered channel that we can send build requests on
var BuildJobQueue = make(chan BuildRequest, BUILD_QUEUE_MAXLENGTH)

func Collector(w http.ResponseWriter, r *http.Request) {
	// This is only allowed with a POST
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Package name
	pkgname := r.FormValue("source-package")
	if pkgname == "" {
		http.Error(w, "You must specify a source package name", http.StatusBadRequest)
		return
	}

	// Allocate a new global id
	id := atomic.AddUint64(&globalRequestId, 1)

	// Create a build request
	request := BuildRequest{
		Id:            id,
		SourcePackage: pkgname,
		Started:       time.Now(),
		Finished:      time.Time{},
		Result:        false,
	}

	// Push it onto the queue
	BuildJobQueue <- request
	logging.Infof("Queued build request #%d (package \"%s\")\n", id, pkgname)

	// Reply
	w.WriteHeader(http.StatusCreated)
}
