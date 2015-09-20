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

var CmdRemovePackage = cli.Command{
	Name:        "remove-package",
	Usage:       "Remove a package",
	Description: `Remove a package from the database.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("name") {
			logging.Errorln("You must specify the package name")
			return ErrWrongArguments
		}
		return nil
	},

	Action: runRemovePackage,
	Flags: []cli.Flag{
		cli.StringFlag{"name, n", "", "package name", ""},
	},
}

func runRemovePackage(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Remove package
	name := ctx.String("name")
	if err = client.RemovePackage(name); err != nil {
		logging.Errorf("Failed to remove package \"%s\": %s\n", name, err)
		return
	}
	logging.Infof("Package \"%s\" removed successfully\n", name)
}
