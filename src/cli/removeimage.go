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
	"google.golang.org/grpc"
)

var CmdRemoveImage = cli.Command{
	Name:        "remove-image",
	Usage:       "Remove a image",
	Description: `Remove a image from the database.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("name") {
			logging.Errorln("You must specify the image name")
			return ErrWrongArguments
		}
		return nil
	},

	Action: runRemoveImage,
	Flags: []cli.Flag{
		cli.StringFlag{"name, n", "", "image name", ""},
	},
}

func runRemoveImage(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Remove image
	name := ctx.String("name")
	if err = client.RemoveImage(name); err != nil {
		logging.Errorf("Failed to remove image \"%s\": %s\n", name, err)
		return
	}
	logging.Infof("Image \"%s\" removed successfully\n", name)
}
