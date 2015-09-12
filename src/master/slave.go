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
	"net"
	"sync/atomic"
)

// Holds the last global slave id
var globalSlaveId uint32 = 0

// Slave structure
type Slave struct {
	Id             uint32
	Name           string
	Channels       []string
	Architectures  []string
	Registered     bool
	Active         bool
	RequestChannel chan BuildRequest
	QueueChannel   chan chan BuildRequest
	QuitChannel    chan bool
}

// Associates client addresses with their Slave
var slaves map[net.Addr]*Slave

// Initialize the slaves map
func init() {
	slaves = make(map[net.Addr]*Slave)
}

// Creates and returns a new Slave object
func NewSlave(name string, chans []string, archs []string) *Slave {
	// Allocate a new global id
	id := atomic.AddUint32(&globalSlaveId, 1)

	// Create and return the object
	slave := &Slave{
		Id:             id,
		Name:           name,
		Channels:       chans,
		Architectures:  archs,
		Registered:     true,
		Active:         true,
		RequestChannel: make(chan BuildRequest),
		QueueChannel:   SlaveQueue,
		QuitChannel:    make(chan bool),
	}
	return slave
}

// Start a goroutine for the slave
func (s *Slave) Start() {
	go func() {
		for {
			// Add to the queue
			s.QueueChannel <- s.RequestChannel

			select {
			case request := <-s.RequestChannel:
				// Receive a build request
				logging.Infof("Build request #%d (package \"%s\") scheduled on \"%s\"\n",
					request.Id, request.SourcePackage, s.Name)
			case <-s.QuitChannel:
				// Slave has been asked to stop
				return
			}
		}
	}()
}

// Ask the slave to stop and, note that it will actually stop
// only after it has finished the job
func (s *Slave) Stop() {
	go func() {
		s.QuitChannel <- true
	}()
}
