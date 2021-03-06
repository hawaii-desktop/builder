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
	"github.com/hawaii-desktop/builder"
)

// Represents a job.
type Job struct {
	// Base.
	*builder.Job
	// Channel.
	Channel chan bool `json:"-"`
}

// Return the slave topic name based in the <type>/<arch> format,
// where type is the job type (package or image) and <arch> the
// architecture (i386, x86_64, ...).
func (j *Job) TopicName() string {
	switch j.Type {
	case builder.JOB_TARGET_TYPE_PACKAGE:
		return "package/" + j.Architecture
	case builder.JOB_TARGET_TYPE_IMAGE:
		return "image/" + j.Architecture
	}

	panic("Unknown job type")
}
