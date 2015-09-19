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
	"gopkg.in/gcfg.v1"
	"os"
	"runtime"
)

const APP_VER = "0.0.0"

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	app := cli.NewApp()
	app.Name = "builder-cli"
	app.Usage = "Command line client for Builder"
	app.Version = APP_VER
	app.Commands = []cli.Command{
		CmdBuild,
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{"config, c", "<filename>", "custom configuration file path", ""},
	}
	app.Before = func(ctx *cli.Context) error {
		// Load the configuration
		var configArg string
		if ctx.IsSet("config") {
			configArg = ctx.String("config")
		} else {
			configArg = "builder-cli.ini"
		}
		return gcfg.ReadFileInto(&Config, configArg)
	}
	app.Run(os.Args)
}
