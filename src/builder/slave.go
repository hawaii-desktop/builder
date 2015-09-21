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
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/hawaii-desktop/builder/src/pidfile"
	"github.com/hawaii-desktop/builder/src/slave"
	"google.golang.org/grpc"
	"gopkg.in/gcfg.v1"
	"os"
	"os/signal"
)

var CmdSlave = cli.Command{
	Name:  "slave",
	Usage: "Perform tasks on a separate machine",
	Description: `Accept build requests from the master matching its configuration
and perform the task assigned.`,
	Action: runSlave,
	Flags: []cli.Flag{
		cli.StringFlag{"name, n", "", "Override slave name from configuration", ""},
		cli.StringFlag{"config, c", "<filename>", "Custom configuration file path", ""},
	},
}

func runSlave(ctx *cli.Context) {
	// Load the configuration
	var configArg string
	if ctx.IsSet("config") {
		configArg = ctx.String("config")
	} else {
		configArg = "builder-slave.ini"
	}
	err := gcfg.ReadFileInto(&slave.Config, configArg)
	if err != nil {
		logging.Fatalln(err)
	}

	// Override configuration
	if ctx.IsSet("name") {
		slave.Config.Slave.Name = ctx.String("name")
	}

	// Sanity check
	if slave.Config.Slave.Name == "" {
		logging.Fatalln("You must specify the slave name")
	}
	if len(slave.Config.Slave.Channels) == 0 {
		logging.Fatalln("You must specify the channels to subscribe")
	}
	if len(slave.Config.Slave.Architectures) == 0 {
		logging.Fatalln("You must specify the supported architectures")
	}

	// Acquire PID file
	if os.Getuid() == 0 {
		pidFile, err := pidfile.New(fmt.Sprintf("/run/builder/slave-%s.pid", slave.Config.Slave.Name))
		if err != nil {
			logging.Fatalf("Unable to create PID file: %s", err.Error())
		}
		err = pidFile.TryLock()
		if err != nil {
			logging.Fatalf("Unable to acquire PID file: %s", err.Error())
		}
		defer pidFile.Unlock()
	}

	// Connect to the master
	conn, err := grpc.Dial(slave.Config.Master.Address, grpc.WithInsecure())
	defer conn.Close()

	// We are finally connected
	logging.Infoln("Connected to master")

	// Client object
	client := slave.NewClient(conn)

	// Subscribe
	err = client.Subscribe()
	if err == nil {
		defer client.Unsubscribe()
	} else {
		logging.Errorf("Unable to register slave: %s\n", err)
		return
	}

	// TODO: We need to register the slave again if the master is restarted

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan

	// Now quit
	logging.Traceln("Quitting...")

	// Close client
	client.Close()
}
