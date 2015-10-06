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
	"bytes"
	"encoding/binary"
	"github.com/boltdb/bolt"
)

// Return a new unique job id.
func (db *Database) NewJobId() uint64 {
	db.jobIdMutex.Lock()
	defer db.jobIdMutex.Unlock()
	db.globalJobId += 1
	db.setLastId("job", db.globalJobId)
	return db.globalJobId
}

// Return a new unique slave id.
func (db *Database) NewSlaveId() uint32 {
	db.slaveIdMutex.Lock()
	defer db.slaveIdMutex.Unlock()
	db.globalSlaveId += 1
	db.setLastId("slave", uint64(db.globalSlaveId))
	return db.globalSlaveId
}

// Return the last job id from the sequence.
func (db *Database) LastJobId() uint64 {
	return db.globalJobId
}

// Return the last slave id from the sequence.
func (db *Database) LastSlaveId() uint32 {
	return db.globalSlaveId
}

// Return the last id from the sequence specified.
func (db *Database) getLastId(name string) (uint64, bool) {
	var result uint64
	err := db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("sequences"))
		if bucket == nil {
			return ErrBucketNotFound
		}

		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			buf := bytes.NewReader(v)
			return binary.Read(buf, binary.LittleEndian, &result)
		}
		return nil
	})
	return result, err == nil
}

// Save the last id for the specified sequence.
func (db *Database) setLastId(name string, val uint64) {
	err := db.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("sequences"))
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, val)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(name), buf.Bytes())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
}
