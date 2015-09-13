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

package protocol

type (
	// Generic message: Type is one of the constants and Data
	// is one of the payload structs
	Message struct {
		Type int
		Data interface{}
	}

	// Registration request (slave -> master)
	RegisterRequest struct {
		Name          string
		Channels      []string
		Architectures []string
	}

	// Registration response (master -> slave)
	RegisterResponse struct {
		Result bool
	}

	// Job request (master -> slave)
	NewJobMessage struct {
		Id            uint64
		SourcePackage string
	}

	// Job finished (slave -> master)
	JobFinishedMessage struct {
		Id     uint64
		Status uint
	}
)

const (
	// Message types (slave -> master)
	MSG_SLAVE_REGISTER    = iota + 1
	MSG_SLAVE_UNREGISTER  = iota + 2
	MSG_SLAVE_PONG        = iota + 3
	MSG_SLAVE_JOBFINISHED = iota + 4

	// Message types (master -> slave)
	MSG_MASTER_REGISTER   = iota + 1
	MSG_MASTER_UNREGISTER = iota + 2
	MSG_MASTER_PING       = iota + 3
	MSG_MASTER_NEWJOB     = iota + 4

	// Job status
	JOB_STATUS_SUCCESSFUL = iota + 10
	JOB_STATUS_FAILED     = iota + 11
)
