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
	"bytes"
	"encoding/gob"
	"net"
)

func init() {
	// Register custom types
	gob.Register(protocol.RegisterRequest{})
	gob.Register(protocol.RegisterResponse{})
}

func encodeData(msg *protocol.Message) {
	enc := gob.NewEncoder(conn)
	err := enc.Encode(msg)
	if err != nil {
		logging.Errorf("Failed to send ping request to %s: %s\n", conn.RemoteAddr(), err.Error())
		return
	}

}

func decodeData(buffer []byte) *protocol.Message {
	dec := gob.NewDecoder(bytes.NewBuffer(buffer))
	msg := &protocol.Message{}
	dec.Decode(msg)
	return msg
}

// Send a ping request to a slave and wait for a pong,
// if it doesn't happen for 3 times the slave is disabled
func pingPong(conn *net.TCPConn) {
	msg := &protocol.Message{protocol.MSG_MASTER_PING, nil}
	enc := gob.NewEncoder(conn)
	err := enc.Encode(msg)
	if err != nil {
		logging.Errorf("Failed to send ping request to %s: %s\n", conn.RemoteAddr(), err.Error())
		return
	}

	buffer := make([]byte, 4096)
	if _, err := conn.Read(buffer); err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			return
		}
		logging.Warningf("Failed to receive pong from %s: %s\n", conn.RemoteAddr(), err.Error())
		return
	}

	slave := slaves[conn.RemoteAddr()]

	reply := decodeData(buffer)
	switch reply.Type {
	case protocol.MSG_SLAVE_PONG:
		slave.Active = true
		break
	default:
		if slave.Active {
			logging.Warningf("Slave \"%s\" (%s) is not feeling well\n", slave.Name)
			slave.Active = false
		}
		break
	}
}

// Process a message received from a client and returns true
// when the goroutine should disconnect from the client
func processMessage(conn *net.TCPConn, msg *protocol.Message) bool {
	switch msg.Type {
	case protocol.MSG_SLAVE_REGISTER:
		payload, ok := msg.Data.(protocol.RegisterRequest)
		if !ok {
			logging.Errorln("Unable to read request from slave")
			return false
		}

		// Do not allow double registrations
		for k, v := range slaves {
			if v.Name == payload.Name && v.Registered {
				logging.Warningf("Slave \"%s\" already registered with client \"%s\"", v.Name, k)
				ok = false
			}
		}

		response := &protocol.Message{protocol.MSG_MASTER_REGISTER, protocol.RegisterResponse{ok}}
		enc := gob.NewEncoder(conn)
		err := enc.Encode(response)
		if err != nil {
			logging.Errorln("Failed to send register reply")
			return false
		}
		if !ok {
			return false
		}

		slaves[conn.RemoteAddr()] = &Slave{
			Name:          payload.Name,
			Channels:      payload.Channels,
			Architectures: payload.Architectures,
			Registered:    true,
			Active:        true,
		}

		logging.Infof("Slave \"%s\" registered for %s", payload.Name, conn.RemoteAddr())
		logging.Infof("\tchannels=%v architectures=%v\n", payload.Channels, payload.Architectures)
		break
	case protocol.MSG_SLAVE_UNREGISTER:
		response := &protocol.Message{protocol.MSG_MASTER_UNREGISTER, nil}
		enc := gob.NewEncoder(conn)
		err := enc.Encode(response)
		if err != nil {
			logging.Errorln("Failed to send unregister reply")
			return false
		}

		for k, v := range slaves {
			if k == conn.RemoteAddr() {
				logging.Infof("Slave \"%s\" unregistered", v.Name)
				slaves[k].Registered = false
				slaves[k].Active = false
				return true
			}
		}
		break
	}

	return false
}
