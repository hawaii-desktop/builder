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

package webserver

import (
	"encoding/json"
	"github.com/hawaii-desktop/builder/src/logging"
)

// A hub mantains the set of active connections and broadcasts
// messages to said connections.
type WebSocketHub struct {
	// Maps connections to their status (register or unregistered).
	connections map[*WebSocketConnection]bool
	// Broadcast channel to send data to the connections.
	broadcast chan []byte
	// Channel of connections that are registering.
	register chan *WebSocketConnection
	// Channel of connections that are unregistering.
	unregister chan *WebSocketConnection
	// Connection registration handlers.
	registerHandlers []ConnectionHandler
	// Connection unregistration handlers.
	unregisterHandlers []ConnectionHandler
}

// A connection handler.
type ConnectionHandler func(c *WebSocketConnection)

// Create a new web socket hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		connections: make(map[*WebSocketConnection]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *WebSocketConnection),
		unregister:  make(chan *WebSocketConnection),
	}
}

// Register a handler to be called when a new connection
// is established.
func (h *WebSocketHub) HandleRegister(f ConnectionHandler) {
	h.registerHandlers = append(h.registerHandlers, f)
}

// Register a handler to be called when a connection is closed.
func (h *WebSocketHub) HandleUnregister(f ConnectionHandler) {
	h.unregisterHandlers = append(h.unregisterHandlers, f)
}

// Start processing connections.
func (h *WebSocketHub) Run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
			for _, handler := range h.registerHandlers {
				handler(c)
			}
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				for _, handler := range h.unregisterHandlers {
					handler(c)
				}

				delete(h.connections, c)
				close(c.send)
				close(c.Outgoing)
			}
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					close(c.Outgoing)
					close(c.send)
					delete(h.connections, c)
				}
			}
		}
	}
}

// Broadcast a message.
func (h *WebSocketHub) Broadcast(msg interface{}) {
	buffer, err := json.Marshal(msg)
	if err != nil {
		logging.Errorf("Unable to marshal message: %s\n", err)
		return
	}
	h.broadcast <- buffer
}
