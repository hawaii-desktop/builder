/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2016 Pier Luigi Fiorini
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
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"time"
)

type Project struct {
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Repos          string    `json:"repos,omitempty"`
	Chroots        []string  `json:"chroots"`
	AutoCreateRepo bool      `json:"auto_createrepo"`
	BuildEnableNet bool      `json:"build_enable_net"`
	WebHookSecret  string    `json:"webhook_secret"`
	Created        time.Time `json:"created"`
}

// Return whether the project was stored into the db.
func (db *Database) HasProject(name string) bool {
	var found bool = false
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("project"))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, _ := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, _ = c.Next() {
			found = true
			return nil
		}
		return nil
	})
	return found
}

// Return a list of project names.
func (db *Database) GetProjectNames() []string {
	var list = []string{}
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("project"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			list = append(list, string(k))
			return nil
		})
		return nil
	})
	return list
}

// Return a list of all projects.
func (db *Database) ListAllProjects() []*Project {
	var list []*Project
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("project"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			prj := &Project{}
			json.Unmarshal(v, &prj)
			list = append(list, prj)
			return nil
		})
		return nil
	})
	return list
}

// Return a project from the database.
func (db *Database) GetProject(name string) *Project {
	var prj *Project = nil
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("project"))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			json.Unmarshal(v, &prj)
			return nil
		}
		return nil
	})
	return prj
}

// Add a project to the database.
func (db *Database) AddProject(prj *Project) error {
	encoded, err := json.Marshal(prj)
	if err != nil {
		return err
	}

	err = db.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("project"))
		if err != nil {
			return err
		}

		now := time.Now().Format(time.RFC3339)
		prj.WebHookSecret = fmt.Sprintf("%x", sha1.Sum([]byte(now)))
		prj.Created = time.Now()

		err = bucket.Put([]byte(prj.Name), encoded)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

// Remove a project from the database.
func (db *Database) RemoveProject(name string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("project"))
		if bucket == nil {
			return nil
		}

		// Unmarshal the project
		prj := &Project{}
		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			if err := json.Unmarshal(v, &prj); err != nil {
				return err
			}
		}

		// Delete the bucket
		if err := bucket.Delete([]byte(name)); err != nil {
			return err
		}

		return nil
	})
}

// Iterate the projects list.
func (db *Database) ForEachProject(f func(prj *Project)) {
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("project"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			var prj *Project
			err := json.Unmarshal(v, &prj)
			if err != nil {
				return err
			}

			f(prj)

			return nil
		})

		return nil
	})
}
