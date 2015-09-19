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

import (
	"fmt"
	"github.com/boltdb/bolt"
	"time"
)

type Database struct {
	db *bolt.DB
}

// Create a new database.
// Do not call this twice with the same path.
func NewDatabase(path string) (*Database, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

// Store key/value into a bucket.
func (d *Database) Store(name []byte, key []byte, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(name)
		if err != nil {
			return err
		}

		err = bucket.Put(key, value)
		if err != nil {
			return err
		}

		return nil
	})
}

// Retrieve a key from a bucket.
func (d *Database) Retrieve(name []byte, key []byte) ([]byte, error) {
	var value []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(name)
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", name)
		}

		value = bucket.Get(key)

		return nil
	})
	return value, err
}

// Close database.
func (d *Database) Close() {
	d.db.Close()
	d.db = nil
}
