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
	"sync"
	"time"
)

type Service struct {
	ch        chan bool
	waitGroup *sync.WaitGroup
}

func NewService() *Service {
	// Create the Service
	s := &Service{
		ch:        make(chan bool),
		waitGroup: &sync.WaitGroup{},
	}
	s.waitGroup.Add(1)
	return s
}

// Accept a connection from a slave and spawn a goroutine to serve it
// and stop listening if nothing is received from the channel
func (s *Service) Serve(listener *net.TCPListener) {
	defer s.waitGroup.Done()

	for {
		select {
		case <-s.ch:
			logging.Infoln("Stop listening on", listener.Addr())
			listener.Close()
			return
		default:
		}

		listener.SetDeadline(time.Now().Add(1e9))

		conn, err := listener.AcceptTCP()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			logging.Warningln(err)
		}

		logging.Traceln("Accepted connection from", conn.RemoteAddr())
		s.waitGroup.Add(1)
		go s.serve(conn)
	}
}

// Stop the service by closing the channel, block until
// it's completely stopped
func (s *Service) Stop() {
	close(s.ch)
	s.waitGroup.Wait()
}

// Serve a connection by reading and writing what was read.  That's right, this
// is an echo service.  Stop reading and writing if anything is received on the
// service's channel but only after writing what was read.
func (s *Service) serve(conn *net.TCPConn) {
	defer conn.Close()
	defer s.waitGroup.Done()

	for {
		select {
		case <-s.ch:
			logging.Traceln("Disconnecting", conn.RemoteAddr())
			return
		default:
		}

		// Do not block on read
		conn.SetDeadline(time.Now().Add(1e9))

		// Read data from the client
		buffer := make([]byte, 4096)
		if _, err := conn.Read(buffer); err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			logging.Warningln(err)
			return
		}

		// Decode and process the message
		msg := decodeData(buffer)
		if processMessage(conn, msg) {
			return
		}
	}
}
