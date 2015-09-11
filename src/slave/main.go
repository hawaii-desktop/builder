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
	"os"
	"os/signal"
)

const (
	LOG_FILENAME   = "slave.log"
	MASTER_ADDRESS = ":9989"
	SLAVE_NAME     = "slave1"
)

var (
	alive         = true
	registered    = false
	channels      = []string{"package", "image"}
	architectures = []string{"i386", "armfp"}
)

func main() {
	// Resolve address
	tcpAddr, err := net.ResolveTCPAddr("tcp", MASTER_ADDRESS)
	if err != nil {
		logging.Fatalf("Failed to resolve %s address: %s\n", MASTER_ADDRESS, err.Error())
	}

	// Connect to the server
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logging.Fatalf("Failed to connect to the master on %s: %s\n", MASTER_ADDRESS, err.Error())
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
