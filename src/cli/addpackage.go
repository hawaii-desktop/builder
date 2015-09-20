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
	"errors"
	"github.com/codegangsta/cli"
	"github.com/hawaii-desktop/builder/src/logging"
	"google.golang.org/grpc"
)

var (
	ErrWrongArguments = errors.New("wrong arguments")
)

var CmdAddPackage = cli.Command{
	Name:        "add-package",
	Usage:       "Add package",
	Description: `Add a package to the database.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("name") {
			logging.Errorln("You must specify the package name")
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
		if ctx.IsSet("ci") {
			if !ctx.IsSet("upstream-vcs") {
				logging.Errorln("You must specify upstream VCS information")
				return ErrWrongArguments
			}
		}
		return nil
	},
	Action: runAddPackage,
	Flags: []cli.Flag{
		cli.StringFlag{"name, n", "<name>", "package name", ""},
		cli.StringFlag{"archs, a", "<arch1>, <arch2>, <archN>...", "supported architectures", ""},
		cli.BoolFlag{"ci", "continuous integration?", ""},
		cli.StringFlag{"vcs", "<url>#branch=<branch>", "packaging VCS", ""},
		cli.StringFlag{"upstream-vcs", "<url>#branch=<branch>", "upstream VCS (only for CI)", ""},
	},
}

func runAddPackage(ctx *cli.Context) {
	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Add a package
	name := ctx.String("name")
	archs := ctx.String("archs")
	ci := ctx.Bool("ci")
	vcs := ctx.String("vcs")
	uvcs := ctx.String("upstream-vcs")
	if !ctx.IsSet("upstream-vcs") {
		uvcs = ""
	}
	if err = client.AddPackage(name, archs, ci, vcs, uvcs); err != nil {
		logging.Errorln(err)
		return
	}
	logging.Infof("Package \"%s\" added successfully\n", name)
}
