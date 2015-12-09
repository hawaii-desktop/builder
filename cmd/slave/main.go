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
	"github.com/hawaii-desktop/builder/logging"
	"github.com/hawaii-desktop/builder/pidfile"
	"github.com/hawaii-desktop/builder/slave"
	"github.com/hawaii-desktop/builder/version"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/gcfg.v1"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"runtime/pprof"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	app := cli.NewApp()
	app.Name = "builder-slave"
	app.Usage = "Perform tasks on a separate machine"
	app.Version = version.Version
	app.Action = runSlave
	app.Flags = []cli.Flag{
		cli.StringFlag{"config, c", "", "Custom configuration file path", ""},
		cli.StringFlag{"cpuprofile", "", "Write CPU profile to file", ""},
	}
	app.Run(os.Args)
}

func runSlave(ctx *cli.Context) {
	// CPU profile
	if ctx.IsSet("cpuprofile") {
		file, err := os.Create(ctx.String("cpuprofile"))
		if err != nil {
			logging.Fatalf("Unable to create \"%s\": %s\n", ctx.String("cpuprofile"), err)
		}
		pprof.StartCPUProfile(file)
		defer pprof.StopCPUProfile()
	}

	// Load the configuration
	var configArg string
	if ctx.IsSet("config") {
		configArg = ctx.String("config")
	} else {
		user, _ := user.Current()
		possible := []string{
			user.HomeDir + "/.config/builder/builder-slave.ini",
			"/etc/builder/builder-slave.ini",
			"builder-slave.ini",
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
	if slave.Config.Slave.Types == "" {
		logging.Fatalln("You must specify the channels to subscribe")
	}
	if slave.Config.Slave.Architectures == "" {
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
	conn, err := grpc.Dial(slave.Config.Master.Address, grpc.WithInsecure(), grpc.WithTimeout(5*time.Second))
	defer conn.Close()

	// We are finally connected
	logging.Infoln("Connected to master")

	// Channel used to close all goroutines for job processing
	waitc := make(chan struct{})

	// Client object
	client := slave.NewClient(conn)

	// Subscribe
	clientCtx, err := client.Subscribe()
	if err != nil {
		logging.Errorf("Unable to register slave: %s\n", err)
		return
	}

	// Unsubscribe and close the connection when quitting
	defer func(ctx context.Context) {
		if conn.State() == grpc.Ready {
			client.Unsubscribe(ctx)
		}
		client.Close()
	}(clientCtx)

	// Pick up jobs from the master
	go func() {
		if err := client.PickJob(clientCtx, waitc); err != nil {
			logging.Errorf("Failed to pick up jobs: %s\n", err)
			return
		}
	}()

	// Channel used to quit the program
	quitc := make(chan bool)

	// Quit when the connection is stopped
	// TODO: We need to register the slave again if the master is restarted
	go func() {
		for {
			if conn.State() != grpc.Ready {
				quitc <- true
				return
			}
		}
	}()

	// Gracefully exit with SIGINT and SIGTERM
	go func() {
		sigchan := make(chan os.Signal, 2)
		signal.Notify(sigchan, os.Interrupt)
		signal.Notify(sigchan, os.Kill)
		<-sigchan
		quitc <- true
	}()

	// Wait for the conditions to quit
	<-quitc

	// Now quit
	logging.Traceln("Quitting...")
	close(waitc)
}
