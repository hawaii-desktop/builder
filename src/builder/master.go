/****************************************************************************
 * This file is part of Builder.
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
	"github.com/codegangsta/cli"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/hawaii-desktop/builder/src/master"
	"github.com/hawaii-desktop/builder/src/pidfile"
	pb "github.com/hawaii-desktop/builder/src/protocol"
	"google.golang.org/grpc"
	"gopkg.in/gcfg.v1"
	"net"
	"net/http"
	"os"
	"os/signal"
)

var CmdMaster = cli.Command{
	Name:  "master",
	Usage: "Collect and dispatch build requests",
	Description: `Collect build requests and dispatch them
to the appropriate slave.`,
	Action: runMaster,
	Flags: []cli.Flag{
		cli.StringFlag{"config, c", "", "Custom configuration file path", ""},
	},
}

func runMaster(ctx *cli.Context) {
	// Load the configuration
	var configArg string
	if ctx.IsSet("config") {
		configArg = ctx.String("config")
	} else {
		possible := []string{
			"~/.config/builder/builder-master.ini",
			"/etc/builder/builder-master.ini",
			"builder-master.ini",
		}
		for _, p := range possible {
			_, err := os.Stat(p)
			if err == nil {
				configArg = p
				break
			}
		}
	}
	if configArg == "" {
		logging.Fatalln("Please specify a configuration file")
	}
	err := gcfg.ReadFileInto(&master.Config, configArg)
	if err != nil {
		logging.Fatalln(err)
	}

	// Acquire PID file
	if os.Getuid() == 0 {
		pidFile, err := pidfile.New("/run/builder/master.pid")
		if err != nil {
			logging.Fatalf("Unable to create PID file: %s", err.Error())
		}
		err = pidFile.TryLock()
		if err != nil {
			logging.Fatalf("Unable to acquire PID file: %s", err.Error())
		}
		defer pidFile.Unlock()
	}

	// Create master service
	masterService, err := master.NewMaster()
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Register RPC server
	rpcListener, err := listenRpc(master.Config.Server.Address)
	if err != nil {
		logging.Errorln(err)
		return
	}
	defer rpcListener.Close()
	grpcServer := grpc.NewServer()
	pb.RegisterBuilderServer(grpcServer, masterService)
	go grpcServer.Serve(rpcListener)

	// Start the dispatcher
	master.StartDispatcher()

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan

	// Close service
	masterService.Close()
}

func listenRpc(address string) (*net.TCPListener, error) {
	// Bind and listen for the master <--> slave protocol
	tcpAddr, err := net.ResolveTCPAddr("tcp", master.Config.Server.Address)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	logging.Infoln("Listening on", listener.Addr())

	return listener, nil
}

func listenHttp(address string) (*net.TCPListener, *http.Server, error) {
	// Bind and listen for the http server
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, nil, err
	}
	logging.Infoln("Listening on", listener.Addr())

	// Create the http server
	server := &http.Server{}

	return listener, server, nil
}
