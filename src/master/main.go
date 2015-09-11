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
	LOG_FILENAME   = "master.log"
	MASTER_ADDRESS = ":9989"
)

type Slave struct {
	Name          string
	Channels      []string
	Architectures []string
	Registered    bool
	Active        bool
}

var (
	slaves map[net.Addr]*Slave
)

func init() {
	slaves = make(map[net.Addr]*Slave)
}

func main() {
	// Bind and listen
	tcpAddr, err := net.ResolveTCPAddr("tcp", MASTER_ADDRESS)
	if err != nil {
		logging.Fatalln(err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logging.Fatalln(err)
	}
	logging.Infoln("Listening on", listener.Addr())

	// Create a service and run it in a goroutine
	service := NewService()
	go service.Serve(listener)

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan
	service.Stop()
}
