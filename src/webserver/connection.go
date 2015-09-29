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
	"github.com/gorilla/websocket"
	"time"
)

// Web socket connection.
type WebSocketConnection struct {
	// Outgoing queue, where messages from the
	// connection are picked up.
	Outgoing chan []byte

	// Web socket connection.
	ws *websocket.Conn

	// Channel to write to the connection.
	send chan []byte

	// Hub.
	hub *WebSocketHub
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Create a new web socket connection.
func NewWebSocketConnection(conn *websocket.Conn, hub *WebSocketHub) *WebSocketConnection {
	return &WebSocketConnection{
		Outgoing: make(chan []byte, 256),
		ws:       conn,
		send:     make(chan []byte, 256),
		hub:      hub,
	}
}

// Write a text message to the connection.
func (c *WebSocketConnection) Write(msg interface{}) error {
	buffer, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = c.write(websocket.TextMessage, buffer)
	if err != nil {
		return err
	}

	return nil
}

// Pump messages from the connection to the hub.
func (c *WebSocketConnection) readPump() {
	// Unregister when we finish reading
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	// Read
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Post message to the queue so it can be processed
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		c.Outgoing <- message
	}
}

// Pump messages from the hub to the connection.
func (c *WebSocketConnection) writePump() {
	// Ping timeout
	ticker := time.NewTicker(pingPeriod)

	// Close the connection when finish
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	// Pump broadcasted messages
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// Write a message of the given type and payload.
func (c *WebSocketConnection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}
