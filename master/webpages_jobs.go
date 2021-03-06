/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
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

package master

import (
	"github.com/plimble/ace"
	"strconv"
)

func WebJobHandler(c *ace.C) {
	data := c.GetAll()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		panic(err)
	}
	data["Id"] = id
	c.HTML("job.html", data)
}

func WebJobsHandler(c *ace.C) {
	c.HTML("jobs.html", c.GetAll())
}

func WebJobsQueuedHandler(c *ace.C) {
	c.HTML("queued.html", c.GetAll())
}

func WebJobsDispatchedHandler(c *ace.C) {
	c.HTML("dispatched.html", c.GetAll())
}

func WebJobsCompletedHandler(c *ace.C) {
	c.HTML("completed.html", c.GetAll())
}

func WebJobsFailedHandler(c *ace.C) {
	c.HTML("failed.html", c.GetAll())
}
