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
)

// Buffered channel that holds the build request
// channels from each slave
var SlaveQueue chan chan BuildRequest

// Start the dispatcher of build requests to slaves
func StartDispatcher() {
	// Initialize the channel
	SlaveQueue = make(chan chan BuildRequest, 100)

	go func() {
		for {
			select {
			case request := <-BuildJobQueue:
				logging.Infof("About to dispatch build request #%d (package \"%s\")\n", request.Id, request.SourcePackage)
				go func() {
					slave := <-SlaveQueue
					logging.Infof("Dispatching build request #%d (package \"%s\")", request.Id, request.SourcePackage)
					slave <- request
				}()
			}
		}
	}()
}
