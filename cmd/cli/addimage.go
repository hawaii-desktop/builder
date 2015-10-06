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
	"google.golang.org/grpc"
)

var CmdAddImage = cli.Command{
	Name:        "add-image",
	Usage:       "Add image",
	Description: `Add an image to the database.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("name") {
			logging.Errorln("You must specify the image name")
			return ErrWrongArguments
		}
		if !ctx.IsSet("descr") {
			logging.Errorln("You must specify a description")
			return ErrWrongArguments
		}
		if !ctx.IsSet("archs") {
			logging.Errorln("You must specify the architectures")
			return ErrWrongArguments
		}
		if !ctx.IsSet("vcs") {
			logging.Errorln("You must specify VCS information")
			return ErrWrongArguments
		}
		return nil
	},
	Action: runAddImage,
	Flags: []cli.Flag{
		cli.StringFlag{"name, n", "", "image name", ""},
		cli.StringFlag{"descr, d", "", "image description", ""},
		cli.StringFlag{"archs, a", "<arch1>, <arch2>, <archN>...", "supported architectures", ""},
		cli.StringFlag{"vcs", "", "VCS (format: <url>#branch=<branch>)", ""},
	},
}

func runAddImage(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Add an image
	name := ctx.String("name")
	descr := ctx.String("descr")
	archs := ctx.String("archs")
	vcs := ctx.String("vcs")
	if err = client.AddImage(name, descr, archs, vcs); err != nil {
		logging.Errorln(err)
		return
	}
	logging.Infof("Image \"%s\" added successfully\n", name)
}
