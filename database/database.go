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

package database

import (
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"path"
	"time"
)

type Database struct {
	// Bolt database.
	db *bolt.DB
}

// Errors
var (
	ErrBucketNotFound = errors.New("bucket not found")
)

// Create and open a database.
func NewDatabase(filename string) (*Database, error) {
	// Open database
	os.MkdirAll(path.Dir(filename), 0700)
	db, err := bolt.Open(filename, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	d := &Database{db}

	// Return
	return d, nil
}

// Close database
func (db *Database) Close() {
	db.db.Close()
	db.db = nil
}
