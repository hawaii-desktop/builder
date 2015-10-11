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
	Id uint64
	// Name.
	Name string
	// Types of targets supported.
	Types []string
	// Supported architectures.
	Architectures []string
	// Whether it has subscribed to the stream or not.
	Subscribed bool
	// Whether it is active or not.
	Active bool
	// Channels to pick up jobs from for each topic.
	jobChannels map[string]chan *Job
	// Channel used to stop processing jobs for each topic.
	quitChannels map[string]chan bool
}

// Creates and returns a new Slave object
func NewSlave(id uint64, name string, types []string, archs []string) *Slave {
	// Create and return the object
	slave := &Slave{
		Id:            id,
		Name:          name,
		Types:         types,
		Architectures: archs,
		Subscribed:    true,
		Active:        true,
		jobChannels:   make(map[string]chan *Job),
		quitChannels:  make(map[string]chan bool),
	}

	// Initialize job channels based on topics
	for _, topic := range slave.Topics() {
		slave.jobChannels[topic] = make(chan *Job)
		slave.quitChannels[topic] = make(chan bool)
	}

	// Return
	return slave
}

// Return the topics that this slave is interested in.
func (s *Slave) Topics() []string {
	var topics []string
	for _, ttype := range s.Types {
		for _, arch := range s.Architectures {
			topics = append(topics, ttype+"/"+arch)
		}
	}
	return topics
}

// Ask the slave to stop after the jobs assigned to it has finished.
func (s *Slave) Stop() {
	s.Active = false

	go func() {
		for _, topic := range s.Topics() {
			s.quitChannels[topic] <- true
		}
	}()
}
