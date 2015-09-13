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
	"../common/pidfile"
	"net"
	"os"
	"os/signal"
)

var (
	alive      = true
	registered = false
)

func main() {
	// Acquire PID file
	pidFile, err := pidfile.New("/tmp/builder/slave.pid")
	if err != nil {
		logging.Fatalf("Unable to create PID file: %s", err.Error())
	}
	err = pidFile.TryLock()
	if err != nil {
		logging.Fatalf("Unable to acquire PID file: %s", err.Error())
	}
	defer pidFile.Unlock()

	// Resolve address
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Master.Address)
	if err != nil {
		logging.Fatalf("Failed to resolve %s address: %s\n", tcpAddr, err.Error())
	}

	// Connect to the server
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logging.Fatalf("Failed to connect to the master on %s: %s\n", tcpAddr, err.Error())
	}

	// Register slave
	registerSlave(conn)

	// Handle protocol
	go func() {
		for alive {
			handleRequest(conn)
		}
	}()

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan

	// Now quit
	alive = false
	logging.Traceln("Quitting...")

	// Unregister slave
	unregisterSlave(conn)

	// Close connection
	conn.Close()
}
