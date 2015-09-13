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
	"../common/protocol"
	"encoding/gob"
	"net"
)

type BuildRequest struct {
	Id            uint64
	SourcePackage string
	Status        uint
	Connection    *net.TCPConn
}

const (
	BUILD_REQUEST_STATUS_PROCESSING = iota
	BUILD_REQUEST_STATUS_SUCCESSFUL = protocol.JOB_STATUS_SUCCESSFUL
	BUILD_REQUEST_STATUS_FAILED     = protocol.JOB_STATUS_FAILED
)

func NewBuildRequest(id uint64, pkgname string, conn *net.TCPConn) *BuildRequest {
	br := &BuildRequest{
		Id:            id,
		SourcePackage: pkgname,
		Status:        BUILD_REQUEST_STATUS_PROCESSING,
		Connection:    conn,
	}
	return br
}

func (br *BuildRequest) Process() {
	// TODO: Fetch sources
	// TODO: Build

	br.Status = BUILD_REQUEST_STATUS_FAILED

	// Notify master
	logging.Infoln("...")
	br.sendFinished()
	logging.Infof("Finished job #%d (target \"%s\")\n", br.Id, br.SourcePackage)
}

func (br *BuildRequest) sendFinished() {
	msg := &protocol.JobFinishedMessage{br.Id, br.Status}
	envelope := &protocol.Message{protocol.MSG_SLAVE_JOBFINISHED, msg}
	enc := gob.NewEncoder(br.Connection)
	err := enc.Encode(envelope)
	if err != nil {
		logging.Errorln("Failed to send job finished:", err.Error())
	}
}
