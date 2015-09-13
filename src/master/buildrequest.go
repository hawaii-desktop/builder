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
	"../common/protocol"
	"time"
)

// Holds the last global request id
var globalRequestId uint64 = 0

type BuildRequest struct {
	Id            uint64
	Slave         *Slave
	SourcePackage string
	Started       time.Time
	Finished      time.Time
	Status        uint
}

const (
	BUILD_REQUEST_STATUS_NOT_STARTED = iota + 1
	BUILD_REQUEST_STATUS_CRASHED     = iota + 2
	BUILD_REQUEST_STATUS_SUCCESSFUL  = protocol.JOB_STATUS_SUCCESSFUL
	BUILD_REQUEST_STATUS_FAILED      = protocol.JOB_STATUS_FAILED
)

func (r BuildRequest) Finish(status uint) {
	r.Status = status
	r.Finished = time.Now()
}
