/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2016 Pier Luigi Fiorini
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

var CmdAddChroot = cli.Command{
	Name:        "add-chroot",
	Usage:       "Add chroot",
	Description: `Add a chroot to the database.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("release") {
			logging.Errorln("You must specify the release")
			return ErrWrongArguments
		}
		if !ctx.IsSet("version") {
			logging.Errorln("You must specify the version")
			return ErrWrongArguments
		}
		if !ctx.IsSet("arch") {
			logging.Errorln("You must specify the architecture")
			return ErrWrongArguments
		}
		return nil
	},
	Action: runAddChroot,
	Flags: []cli.Flag{
		cli.StringFlag{"release, r", "<release>", "release (fedora, epel, ...)", ""},
		cli.StringFlag{"version, v", "<version>", "version (22, 23, rawhide, ...)", ""},
		cli.StringFlag{"arch, a", "<arch>", "architecture (i386, x86_64, armhfp, ...)", ""},
	},
}

func runAddChroot(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Add a chroot
	r := ctx.String("release")
	v := ctx.String("version")
	a := ctx.String("arch")
	if err = client.AddChroot(r, v, a); err != nil {
		logging.Errorln(err)
		return
	}
	logging.Infof("Chroot \"%s-%s-%s\" added successfully\n", r, v, a)
}
