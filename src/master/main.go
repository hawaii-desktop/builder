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
	"net/http"
	"os"
	"os/signal"
)

func listenTcp() *net.TCPListener {
	// Bind and listen for the master <--> slave protocol
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Server.Address)
	if err != nil {
		logging.Fatalln(err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logging.Fatalln(err)
	}
	logging.Infoln("Listening on", listener.Addr())

	return listener
}

func listenHttp() (*net.TCPListener, *http.Server) {
	// Bind and listen for the http server
	tcpAddr, err := net.ResolveTCPAddr("tcp", config.Server.HttpAddress)
	if err != nil {
		logging.Fatalln(err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logging.Fatalln(err)
	}
	logging.Infoln("Listening on", listener.Addr())

	// Create the http server
	server := &http.Server{}

	return listener, server
}

func main() {
	// Acquire PID file
	pidFile, err := pidfile.New("/tmp/builder/master.pid")
	if err != nil {
		logging.Fatalf("Unable to create PID file: %s", err.Error())
	}
	err = pidFile.TryLock()
	if err != nil {
		logging.Fatalf("Unable to acquire PID file: %s", err.Error())
	}
	defer pidFile.Unlock()

	// Protocol between master and slave
	tcpListener := listenTcp()
	service := NewService()
	go service.Serve(tcpListener)

	// Start the dispatcher
	StartDispatcher()

	// HTTP server for dashboard and collector
	httpListener, httpServer := listenHttp()
	http.HandleFunc("/collector", Collector)
	go httpServer.Serve(httpListener)

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan
	service.Stop()
}
