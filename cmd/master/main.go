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
	"github.com/hawaii-desktop/builder/logging"
	"github.com/hawaii-desktop/builder/master"
	"github.com/hawaii-desktop/builder/pidfile"
	pb "github.com/hawaii-desktop/builder/protocol"
	"github.com/hawaii-desktop/builder/version"
	"github.com/hawaii-desktop/builder/webserver"
	"github.com/plimble/ace"
	"google.golang.org/grpc"
	"gopkg.in/gcfg.v1"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"runtime/pprof"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	app := cli.NewApp()
	app.Name = "builder-master"
	app.Usage = "Collect and dispatch build requests"
	app.Version = version.Version
	app.Action = runMaster
	app.Flags = []cli.Flag{
		cli.StringFlag{"config, c", "", "Custom configuration file path", ""},
		cli.StringFlag{"cpuprofile", "", "Write CPU profile to file", ""},
	}
	app.Run(os.Args)
}

func runMaster(ctx *cli.Context) {
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

	// Web server
	webServer := webserver.New(master.Config.Server.HttpAddress)
	webServer.Router.Use(func(c *ace.C) {
		session := c.Sessions("authentication")
		c.Set("IsLoggedIn", session.GetBool("IsLoggedIn", false))
		c.Set("UserName", session.GetString("UserName", ""))
		c.Next()
	})
	webServer.Router.HtmlTemplate(webserver.TemplateRenderer(master.Config.Web.TemplateDir))
	webServer.Router.GET("/", func(c *ace.C) { c.HTML("overview.html", c.GetAll()) })
	webServer.Router.GET("/users/login", master.LoginHandler)
	webServer.Router.GET("/users/logout", master.LogoutHandler)
	webServer.Router.GET("/sso/github", master.SsoGitHubHandler)
	webServer.Router.GET("/job/:id", master.WebJobHandler)
	webServer.Router.GET("/jobs", master.WebJobsHandler)
	webServer.Router.GET("/jobs/queued", master.WebJobsQueuedHandler)
	webServer.Router.GET("/jobs/dispatched", master.WebJobsDispatchedHandler)
	webServer.Router.GET("/jobs/completed", master.WebJobsCompletedHandler)
	webServer.Router.GET("/jobs/failed", master.WebJobsFailedHandler)
	webServer.Router.Static("/css", http.Dir(master.Config.Web.StaticDir+"/css"))
	webServer.Router.Static("/js", http.Dir(master.Config.Web.StaticDir+"/js"))
	webServer.Router.Static("/img", http.Dir(master.Config.Web.StaticDir+"/img"))
	webServer.Router.Static("/repo/packages", http.Dir(master.Config.Storage.RepositoryDir))
	webServer.Router.Static("/repo/images", http.Dir(master.Config.Storage.ImagesDir))
	go func() {
		err = webServer.ListenAndServe()
		if err != nil {
			logging.Fatalln(err)
		}
	}()
	logging.Infoln("Web server listening on", webServer.Address())

	// Create the main object
	m, err := master.NewMaster(webServer.Hub)
	if err != nil {
		logging.Errorln(err)
		return
	}
	defer m.Close()

	// Create storage
	if err := m.CreateStorage(); err != nil {
		logging.Errorln(err)
		return
	}

	// Process repodata updates
	m.ProcessRepoDataUpdates()

	// Handle web socket registration and unregistration
	webServer.Hub.HandleRegister(m.WebSocketConnectionRegistration)
	webServer.Hub.HandleUnregister(m.WebSocketConnectionUnregistration)

	// Create master service
	service := master.NewRpcService(m)

	// Register RPC server
	rpcListener, err := listenRpc(master.Config.Server.Address)
	if err != nil {
		logging.Errorln(err)
		return
	}
	defer rpcListener.Close()
	grpcServer := grpc.NewServer()
	pb.RegisterBuilderServer(grpcServer, service)
	go grpcServer.Serve(rpcListener)

	// Prepare topics
	m.PrepareTopics()

	// Start processing
	go m.Dispatch()
	go m.DeliverWebSocketEvents()

	// Queue jobs that were not picked up by any slave
	// on a previous run
	m.LoadDatabaseJobs()

	// Gracefully exit with SIGINT and SIGTERM
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, os.Kill)
	<-sigchan
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
