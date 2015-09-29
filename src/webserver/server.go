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
	"github.com/gorilla/websocket"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/plimble/ace"
	"github.com/plimble/sessions/store/cookie"
	"net/http"
)

// Web server.
type WebServer struct {
	address  string
	upgrader *websocket.Upgrader
	Router   *ace.Ace
	Hub      *WebSocketHub
}

// Create routing and start a Web server.
func New(address string) *WebServer {
	ws := &WebServer{
		address: address,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		Router: ace.New(),
		Hub:    NewWebSocketHub(),
	}
	store := cookie.NewCookieStore()
	ws.Router.Use(ace.Session(store, nil))
	ws.Router.Use(func(c *ace.C) {
		c.Set("SiteUrl", address)
		c.Set("SiteHost", c.Request.Host)
		c.Next()
	})
	ws.Router.HtmlTemplate(TemplateRenderer())
	ws.Router.Panic(func(c *ace.C, rcv interface{}) {
		switch err := rcv.(type) {
		case error:
			c.String(500, "%s\n%s", err, ace.Stack())
			logging.Errorf("Request %s %s failed: %s\n", c.Request.Method, c.Request.URL, err)
		}
	})
	ws.Router.GET("/ws", ws.webSocket)
	go ws.Hub.Run()
	return ws
}

// Return the network address the web server is listening to.
func (ws *WebServer) Address() string {
	return ws.address
}

// Listen and serve.
func (ws *WebServer) ListenAndServe() error {
	return http.ListenAndServe(ws.address, ws.Router)
}

// Web socket entry point.
func (ws *WebServer) webSocket(c *ace.C) {
	// Sanity check
	if c.Request.Header.Get("Origin") != "http://"+c.Request.Host {
		http.Error(c.Writer, "Origin not allowed", 403)
		return
	}
	if c.Request.URL.Path != "/ws" {
		http.Error(c.Writer, "Not found", 404)
		return
	}
	if c.Request.Method != "GET" {
		http.Error(c.Writer, "Method not allowed", 405)
		return
	}

	// Upgrade
	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Errorf("Failed to open web socket connection: %s\n", err)
		return
	}

	// Handle connection
	wsConn := NewWebSocketConnection(conn, ws.Hub)
	ws.Hub.register <- wsConn
	go wsConn.writePump()
	wsConn.readPump()
}
