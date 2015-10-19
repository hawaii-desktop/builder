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

package slave

import (
	"golang.org/x/net/context"
)

// Data stored in the context.
type SlaveData struct {
	// Identifier for this slave, attributed after subscription.
	// Its value is 0 when unsubscribed.
	Id uint64
	// Main repository path on master.
	MainRepoDir string
	// Staging repository path on master.
	StagingRepoDir string
	// Images repository path on master
	ImagesDir string
	// Repository base URL.
	RepoBaseUrl string
}

// key is an unexported type for keys defined in this package.
type key int

// dataKey is the key for slave.SlaveData values in Contexts.
var dataKey key = 0

// NewContext returns a new Context that carries value d.
func NewContext(ctx context.Context, d *SlaveData) context.Context {
	return context.WithValue(ctx, dataKey, d)
}

// FromContext returns the SlaveData value stored in ctx, if any.
func FromContext(ctx context.Context) (*SlaveData, bool) {
	d, ok := ctx.Value(dataKey).(*SlaveData)
	return d, ok
}
