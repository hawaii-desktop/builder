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
	"../common/protocol"
	"errors"
	"net"
	"sync/atomic"
	"time"
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
	Connection     *net.TCPConn
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
func NewSlave(name string, chans []string, archs []string, conn *net.TCPConn) *Slave {
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
		Connection:     conn,
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
			// Do not queue a slave that suddenly unregisters itself
			if !s.Registered || !s.Active {
				return
			}

			// Add to the queue
			s.QueueChannel <- s.RequestChannel

			select {
			case request := <-s.RequestChannel:
				// Send the build request to the slave
				request.Slave = s
				logging.Infof("Build request #%d (package \"%s\") running on \"%s\"\n",
					request.Id, request.SourcePackage, s.Name)
				_ = s.send(request)
				logging.Infof("Finished build request #%d (package \"%s\") on \"%s\"\n",
					request.Id, request.SourcePackage, request.Slave.Name)
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

// Send a build request to the slave
func (s *Slave) send(request BuildRequest) error {
	// Send request
	s.Connection.SetDeadline(time.Now().Add(1e9))
	msg := &protocol.NewJobMessage{request.Id, request.SourcePackage}
	envelope := &protocol.Message{protocol.MSG_MASTER_NEWJOB, msg}
	err := encodeData(s.Connection, envelope)
	if err != nil {
		logging.Errorf("Failed to send new job request to %s: %s\n", s.Connection.RemoteAddr(), err.Error())
		request.Finish(BUILD_REQUEST_STATUS_CRASHED)
		return err
	}

	// Receive reply
	buffer := make([]byte, 4096)
	if _, err := s.Connection.Read(buffer); err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			return err
		}
		logging.Warningf("Failed to receive job finished reply from %s: %s\n", s.Connection.RemoteAddr(), err.Error())
		return err
	}

	// Decode reply
	reply := decodeData(buffer)
	payload, ok := reply.Data.(protocol.JobFinishedMessage)
	if !ok {
		logging.Errorln("Failed to decode job finished reply from", s.Connection.RemoteAddr())
		return errors.New("decoding failed")
	}

	// Save request information and stop
	request.Finish(payload.Status)

	return nil
}
