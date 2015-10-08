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
	"encoding/json"
	"github.com/boltdb/bolt"
	"github.com/hawaii-desktop/builder"
	"strconv"
	"time"
)

type Job struct {
	Id           uint64                `json:"id"`
	Type         builder.JobTargetType `json:"type"`
	Target       string                `json:"target"`
	Architecture string                `json:"arch"`
	Started      time.Time             `json:"started"`
	Finished     time.Time             `json:"finished"`
	Status       builder.JobStatus     `json:"status"`
}

// Return a stored job.
func (db *Database) GetJob(id uint64) *Job {
	var job *Job = nil
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("job"))
		if bucket == nil {
			return nil
		}

		sid := strconv.FormatUint(id, 10)

		c := bucket.Cursor()
		for k, v := c.Seek([]byte(sid)); bytes.Equal(k, []byte(sid)); k, v = c.Next() {
			json.Unmarshal(v, &job)
			return nil
		}
		return nil
	})
	return job
}

// Return a list of jobs that match a certain criteria.
func (db *Database) FilterJobs(filter func(job *Job) bool) []*Job {
	var list []*Job
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("job"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			var job *Job
			err := json.Unmarshal(v, &job)
			if err != nil {
				return err
			}

			if filter(job) {
				list = append(list, job)
			}
			return nil
		})

		return nil
	})
	return list
}

// Iterate the jobs list.
func (db *Database) ForEachJob(f func(job *Job)) {
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("job"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			var job *Job
			err := json.Unmarshal(v, &job)
			if err != nil {
				return err
			}

			f(job)

			return nil
		})

		return nil
	})
}

// Store a job.
func (db *Database) SaveJob(job *Job) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		// Find bucket and return an error if it doesn't exist
		bucket, err := tx.CreateBucketIfNotExists([]byte("job"))
		if err != nil {
			return err
		}

		// Encode
		encoded, err := json.Marshal(job)
		if err != nil {
			return err
		}

		// Save
		sid := strconv.FormatUint(job.Id, 10)
		err = bucket.Put([]byte(sid), encoded)
		if err != nil {
			return err
		}

		return nil
	})
}
