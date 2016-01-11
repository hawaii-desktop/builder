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

var CmdBuild = cli.Command{
	Name:        "build",
	Usage:       "Build packages and images",
	Description: `Request the build of a package or an image.`,
	Before: func(ctx *cli.Context) error {
		if ctx.IsSet("type") {
			t := ctx.String("type")
			if t != "package" && t != "image" {
				logging.Errorf("Wrong target type \"%s\": must be either \"package\" or \"image\"\n", t)
				return ErrWrongArguments
			}
		} else {
			logging.Errorln("You must specify the target type")
			return ErrWrongArguments
		}
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
	Action: runBuild,
	Flags: []cli.Flag{
		cli.StringFlag{"type, t", "", "type (package or image)", ""},
		cli.StringFlag{"name, n", "", "package name", ""},
		cli.StringFlag{"arch, a", "", "architecture", ""},
	},
}

func runBuild(ctx *cli.Context) {
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
	t := ctx.String("type")
	name := ctx.String("name")
	arch := ctx.String("arch")
	var id uint64
	if id, err = client.SendJob(name, arch, t); err != nil {
		logging.Errorln(err)
		return
	}
	logging.Infof("Target \"%s\" build for %s queued as #%d\n", name, arch, id)
}
