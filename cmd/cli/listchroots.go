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

var CmdListChroots = cli.Command{
	Name:        "list-chroots",
	Usage:       "List chroots",
	Description: `List chroots added to the database.`,
	Action:      runListChroots,
	Flags: []cli.Flag{
		cli.BoolFlag{"active, a", "list only active chroots", ""},
		cli.BoolFlag{"inactive, i", "list only inactive chroots", ""},
	},
}

func runListChroots(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// List chroots
	flags := AllChroots
	if ctx.Bool("active") {
		flags = ActiveChroots
	} else if ctx.Bool("inactive") {
		flags = InactiveChroots
	}
	if err = client.ListChroots(flags); err != nil {
		logging.Errorln(err)
		return
	}
}
