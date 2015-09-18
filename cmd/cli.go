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

package cmd

import (
	"github.com/codegangsta/cli"
	bcli "github.com/hawaii-desktop/builder/cli"
	"github.com/hawaii-desktop/builder/common/logging"
	"google.golang.org/grpc"
	"gopkg.in/gcfg.v1"
)

var CmdCli = cli.Command{
	Name:        "cli",
	Usage:       "Queue jobs",
	Description: `Simple command line tool used to enqueue jobs.`,
	Action:      runCli,
	Flags: []cli.Flag{
		cli.StringFlag{"config, c", "<filename>", "Custom configuration file path", ""},
		cli.StringFlag{"build", "<name>", "Build a package", ""},
	},
}

func runCli(ctx *cli.Context) {
	// Check arguments
	if !ctx.IsSet("target") {
		logging.Fatalln("You must specify a target")
	}

	// Load the configuration
	var configArg string
	if ctx.IsSet("config") {
		configArg = ctx.String("config")
	} else {
		configArg = "builder-cli.ini"
	}
	err := gcfg.ReadFileInto(&bcli.Config, configArg)
	if err != nil {
		logging.Fatalln(err)
	}

	// Connect to the master
	conn, err := grpc.Dial(bcli.Config.Master.Address, grpc.WithInsecure())
	defer conn.Close()

	// Create client proxy
	client := bcli.NewClient(conn)

	// Build a package
	if ctx.IsSet("build") {
		pkgname := ctx.String("build")
		var id uint64
		if id, err = client.SendJob(pkgname); err != nil {
			logging.Errorln(err)
			return
		}
		logging.Infof("Package \"%s\" build queued as #%d\n", pkgname, id)
	}
}
