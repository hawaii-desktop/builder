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
	"github.com/hawaii-desktop/builder/src/api"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/hawaii-desktop/builder/src/pidfile"
	pb "github.com/hawaii-desktop/builder/src/protocol"
	"github.com/hawaii-desktop/builder/src/webserver"
	"github.com/plimble/ace"
	"google.golang.org/grpc"
	"gopkg.in/gcfg.v1"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	app := cli.NewApp()
	app.Name = "builder-master"
	app.Usage = "Collect and dispatch build requests"
	app.Version = api.APP_VER
	app.Action = runMaster
	app.Flags = []cli.Flag{
		cli.StringFlag{"config, c", "", "Custom configuration file path", ""},
	}
	app.Run(os.Args)
}

func runMaster(ctx *cli.Context) {
	// Load the configuration
	var configArg string
	if ctx.IsSet("config") {
		configArg = ctx.String("config")
	} else {
		user, _ := user.Current()
		possible := []string{
			user.HomeDir + "/.config/builder/builder-master.ini",
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
	err := gcfg.ReadFileInto(&Config, configArg)
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

	// Web server
	webServer := webserver.New(Config.Server.HttpAddress)
	webServer.Router.HtmlTemplate(webserver.TemplateRenderer(Config.Web.TemplateDir))
	webServer.Router.GET("/", func(c *ace.C) { c.HTML("overview.html", c.GetAll()) })
	webServer.Router.GET("/queued", func(c *ace.C) { c.HTML("queued.html", c.GetAll()) })
	webServer.Router.GET("/completed", func(c *ace.C) { c.HTML("completed.html", c.GetAll()) })
	webServer.Router.GET("/failed", func(c *ace.C) { c.HTML("failed.html", c.GetAll()) })
	webServer.Router.Static("/css", http.Dir(Config.Web.StaticDir+"/css"))
	webServer.Router.Static("/js", http.Dir(Config.Web.StaticDir+"/js"))
	webServer.Router.Static("/img", http.Dir(Config.Web.StaticDir+"/img"))
	go func() {
		err = webServer.ListenAndServe()
		if err != nil {
			logging.Fatalln(err)
		}
	}()
	logging.Infoln("Web server listening on", webServer.Address())

	// Create the main object
	master := NewMaster(webServer.Hub)

	// Handle web socket registration and unregistration
	webServer.Hub.HandleRegister(func(c *webserver.WebSocketConnection) {
		// Send statistics as soon as a client connects
		master.SendStats()
	})
	webServer.Hub.HandleUnregister(func(c *webserver.WebSocketConnection) {
	})

	// Create master service
	service, err := NewRpcService(master)
	if err != nil {
		logging.Errorln(err)
		return
	}
	defer service.Close()

	// Register RPC server
	rpcListener, err := listenRpc(Config.Server.Address)
	if err != nil {
		logging.Errorln(err)
		return
	}
	defer rpcListener.Close()
	grpcServer := grpc.NewServer()
	pb.RegisterBuilderServer(grpcServer, service)
	go grpcServer.Serve(rpcListener)

	// Start processing
	master.Process()

	// Calculate statisti
	service.calculateStats()

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan
}

func listenRpc(address string) (*net.TCPListener, error) {
	// Bind and listen for the master <--> slave protocol
	tcpAddr, err := net.ResolveTCPAddr("tcp", Config.Server.Address)
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
