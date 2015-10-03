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

type statistics struct {
	Queued     int `json:"queued"`
	Dispatched int `json:"dispatched"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Staging    int `json:"staging"`
	Main       int `json:"main"`
}

// Broadcast queue for the web socket.
var WebSocketQueue chan interface{}

// Statistics.
var stats *statistics

func init() {
	stats = &statistics{0, 0, 0, 0, 0, 0}
	WebSocketQueue = make(chan interface{}, 5)
}
