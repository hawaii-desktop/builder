/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
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
	"google.golang.org/grpc"
)

var CmdBuildPackage = cli.Command{
	Name:        "build-package",
	Usage:       "Build packages",
	Description: `Request to build a package.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("name") {
			logging.Errorln("You must specify the target name")
			return ErrWrongArguments
		}
		if !ctx.IsSet("arch") {
			logging.Errorln("You must specify the target architecture")
			return ErrWrongArguments
		}

		return nil
	},
	Action: runBuildPackage,
	Flags: []cli.Flag{
		cli.StringFlag{"name, n", "", "package name", ""},
		cli.StringFlag{"arch, a", "", "architecture", ""},
	},
}

func runBuildPackage(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Build the target
	name := ctx.String("name")
	arch := ctx.String("arch")
	var id uint64
	if id, err = client.SendJob(name, arch, "package"); err != nil {
		logging.Errorln(err)
		return
	}
	logging.Infof("Package \"%s\" build for %s queued as #%d\n", name, arch, id)
}
