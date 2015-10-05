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
	"github.com/plimble/ace"
)

func webJobsHandler(c *ace.C) {
	c.HTML("jobs.html", c.GetAll())
}

func webJobsQueuedHandler(c *ace.C) {
	c.HTML("queued.html", c.GetAll())
}

func webJobsDispatchedHandler(c *ace.C) {
	c.HTML("dispatched.html", c.GetAll())
}

func webJobsCompletedHandler(c *ace.C) {
	c.HTML("completed.html", c.GetAll())
}

func webJobsFailedHandler(c *ace.C) {
	c.HTML("failed.html", c.GetAll())
}
