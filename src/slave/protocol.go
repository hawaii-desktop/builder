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
	"io"
	"net"
	"time"
)

func init() {
	// Register custom types
	gob.Register(protocol.RegisterRequest{})
	gob.Register(protocol.RegisterResponse{})
}

func readFromMaster(conn *net.TCPConn) *protocol.Message {
	// Set deadline before reading
	conn.SetDeadline(time.Now().Add(1e9))

	// Read data from master
	buffer := make([]byte, 4096)
	if _, err := conn.Read(buffer); err != nil {
		if err == io.EOF {
			logging.Fatalln("Master connection went away, exiting...")
		} else {
			if opErr, ok := err.(*net.OpError); ok && !opErr.Timeout() {
				logging.Errorln(err)
			}
		}
		return nil
	}

	// Decode request
	dec := gob.NewDecoder(bytes.NewBuffer(buffer))
	msg := &protocol.Message{}
	dec.Decode(msg)
	return msg
}

func registerSlave(conn *net.TCPConn) {
	// Encode data
	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	payload := &protocol.RegisterRequest{SLAVE_NAME, channels, architectures}
	msg := &protocol.Message{protocol.MSG_SLAVE_REGISTER, payload}
	err := enc.Encode(msg)
	if err != nil {
		logging.Fatalln("Failed to register to the master:", err)
	}

	// Send registration message
	conn.SetDeadline(time.Now().Add(1e9))
	if _, err = conn.Write(buffer.Bytes()); err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			logging.Fatalln("Timeout during slave registration")
		} else if err == io.EOF {
			logging.Fatalln("Master connection went away during slave registration")
		} else {
			logging.Fatalln(err)
		}
		return
	}

	// Read reply
	msg = readFromMaster(conn)
	if msg != nil && msg.Type == protocol.MSG_MASTER_REGISTER {
		registered = true
		logging.Infoln("Slave registered successfully")
	}
}

func unregisterSlave(conn *net.TCPConn) {
	// Encode data
	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	msg := &protocol.Message{protocol.MSG_SLAVE_UNREGISTER, nil}
	err := enc.Encode(msg)
	if err != nil {
		logging.Errorln("Failed to unregister from the master:", err.Error())
		return
	}

	// Send unregistration message
	conn.SetDeadline(time.Now().Add(1e9))
	if _, err = conn.Write(buffer.Bytes()); err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			logging.Fatalln("Timeout during slave unregistration")
		} else if err == io.EOF {
			logging.Fatalln("Master connection went away during slave unregistration")
		} else {
			logging.Fatalln(err)
		}
		return
	}

	// Read reply
	msg = readFromMaster(conn)
	if msg != nil && msg.Type == protocol.MSG_MASTER_UNREGISTER {
		registered = false
		logging.Infoln("Slave unregistered successfully")
	}
}

func handleRequest(conn *net.TCPConn) {
	// Read from master
	msg := readFromMaster(conn)
	if msg == nil {
		return
	}

	// Parse reply
	switch msg.Type {
	case protocol.MSG_MASTER_UNREGISTER:
		alive = false
		break
	case protocol.MSG_MASTER_PING:
		enc := gob.NewEncoder(conn)
		reply := &protocol.Message{protocol.MSG_SLAVE_PONG, nil}
		err := enc.Encode(reply)
		if err != nil {
			logging.Errorln("Failed to send pong reply:", err.Error())
		}
		break
	}
}
