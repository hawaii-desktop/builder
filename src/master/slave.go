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

package master

// Slave structure
type Slave struct {
	// Identifier.
	Id uint32
	// Name.
	Name string
	// Channels to subscribe to.
	Channels []string
	// Supported architectures.
	Architectures []string
	// Whether it has subscribed to the stream or not.
	Subscribed bool
	// Whether it is active or not.
	Active bool
	// Channel to pick up jobs from.
	JobChannel chan *Job
	// Buffered channel for jobs.
	QueueChannel chan chan *Job
	// Channel used to stop processing jobs.
	QuitChannel chan bool
}

// Creates and returns a new Slave object
func NewSlave(id uint32, name string, chans []string, archs []string) *Slave {
	// Create and return the object
	slave := &Slave{
		Id:            id,
		Name:          name,
		Channels:      chans,
		Architectures: archs,
		Subscribed:    true,
		Active:        true,
		JobChannel:    make(chan *Job),
		QueueChannel:  SlaveQueue,
		QuitChannel:   make(chan bool),
	}
	return slave
}

// Ask the slave to stop after the jobs assigned to it has finished.
func (s *Slave) Stop() {
	s.Active = false

	go func() {
		s.QuitChannel <- true
	}()
}
