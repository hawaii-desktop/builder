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
)

type Image struct {
	Name          string   `json:"name"`
	Description   string   `json:"descr"`
	Architectures []string `json:"archs"`
	Vcs           VcsInfo  `json:"vcs"`
}

// Return whether the image was stored into the db.
func (db *Database) HasImage(name string) bool {
	var found bool = false
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("image"))
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

// Return a list of image names.
func (db *Database) GetImageNames() []string {
	var list = []string{}
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("image"))
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

// Return a list of all images.
func (db *Database) ListAllImages() []*Image {
	var list []*Image
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("image"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			img := &Image{}
			json.Unmarshal(v, &img)
			list = append(list, img)
			return nil
		})
		return nil
	})
	return list
}

// Return an image from the database.
func (db *Database) GetImage(name string) *Image {
	var img *Image = nil
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("image"))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			json.Unmarshal(v, &img)
			return nil
		}
		return nil
	})
	return img
}

// Add an image to the database.
func (db *Database) AddImage(img *Image) error {
	encoded, err := json.Marshal(img)
	if err != nil {
		return nil
	}
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("image"))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(img.Name), encoded)
		if err != nil {
			return err
		}

		return nil
	})
}

// Remove an image from the database.
func (db *Database) RemoveImage(name string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("image"))
		if bucket == nil {
			return nil
		}

		err := bucket.Delete([]byte(name))
		if err != nil {
			return err
		}

		return nil
	})
}
