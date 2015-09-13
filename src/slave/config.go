/****************************************************************************
 * This file is part of Hawaii.
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
	"../common/logging"
	"gopkg.in/gcfg.v1"
)

type Config struct {
	Master struct {
		Address string
	}
	Slave struct {
		Name          string
		Channels      []string
		Architectures []string
	}
}

var config Config

func init() {
	err := gcfg.ReadFileInto(&config, "slave.cfg")
	if err != nil {
		logging.Fatalln(err)
	}

	if config.Master.Address == "" {
		logging.Fatalln("You must specify the master address")
	}
	if config.Slave.Name == "" {
		logging.Fatalln("You must specify the slave name")
	}
	if len(config.Slave.Channels) == 0 {
		logging.Fatalln("You must specify the channels to subscribe")
	}
	if len(config.Slave.Architectures) == 0 {
		logging.Fatalln("You must specify the supported architectures")
	}
}
