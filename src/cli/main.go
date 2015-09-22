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
	"gopkg.in/gcfg.v1"
	"os"
	"runtime"
)

const APP_VER = "0.0.0"

var (
	ErrWrongArguments = errors.New("wrong arguments")
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	app := cli.NewApp()
	app.Name = "builder-cli"
	app.Usage = "Command line client for Builder"
	app.Version = APP_VER
	app.Commands = []cli.Command{
		CmdAddPackage,
		CmdRemovePackage,
		CmdListPackages,
		CmdAddImage,
		CmdRemoveImage,
		CmdListImages,
		CmdImport,
		CmdBuild,
		CmdCert,
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{"config, c", "", "custom configuration file path", ""},
		cli.StringFlag{"address, a", "", "override master address from the configuration file", ""},
	}
	app.Before = func(ctx *cli.Context) error {
		// Load the configuration
		var configArg string
		if ctx.IsSet("config") {
			configArg = ctx.String("config")
		} else {
			possible := []string{
				"~/.config/builder/builder-cli.ini",
				"/etc/builder/builder-cli.ini",
				"builder-cli.ini",
			}
			for _, p := range possible {
				_, err := os.Stat(p)
				if err == nil {
					configArg = p
					break
				}
			}
		}
		if configArg == "" {
			logging.Fatalln("Please specify a configuration file")
		}
		err := gcfg.ReadFileInto(&Config, configArg)
		if err != nil {
			return err
		}

		// Change master address if requested
		if ctx.IsSet("address") {
			Config.Master.Address = ctx.String("address")
		}

		return nil
	}
	app.Run(os.Args)
}
