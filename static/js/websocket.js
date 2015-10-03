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

var WEB_SOCKET_STATISTICS = 0;
var WEB_SOCKET_QUEUED_JOBS = 1;
var WEB_SOCKET_DISPATCHED_JOBS = 2;
var WEB_SOCKET_COMPLETED_JOBS = 3;
var WEB_SOCKET_FAILED_JOBS = 4;

function createWebSocket(address, processFunc) {
    wsConn = new WebSocket(address);
    wsConn.onopen = function(event) {
        // Reset the attempts since a new connection just opened
        wsAttempts = 1;

        // Hide dialog
        $("#reconnectWaitDialog").modal("hide");
    };
    wsConn.onclose = function(event) {
        var time = determineInterval(wsAttempts);
        setTimeout(function() {
            // Increment the number of attempts
            wsAttempts++;

            // Show dialog
            $("#reconnectWaitDialog").modal("show");

            // Reconnect
            createWebSocket(address, processFunc);
        }, time);
    };
    wsConn.onmessage = function(event) {
        var obj = $.parseJSON(event.data);
        processFunc(obj);
    };
    wsConn.onerror = function(event) {
    };
}

function determineInterval(k) {
    // Calculate the maximum interval
    var maxInterval = (Math.pow(2, k) - 1) * 1000;

    // Cap the maximum interval to 30s
    if (maxInterval > 30*1000)
        maxInterval = 30*1000;

    // Generate a random interval
    return Math.random() * maxInterval;
}

// vim: set noai ts=4 sw=4 expandtab:
