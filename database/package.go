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

type VcsInfo struct {
	Url    string `json:"url"`
	Branch string `json:"branch"`
}

type Package struct {
	Name          string   `json:"name"`
	Architectures []string `json:"architectures"`
	Ci            bool     `json:"ci"`
	Vcs           VcsInfo  `json:"vcs"`
	UpstreamVcs   VcsInfo  `json:"upstream_vcs"`
}

// Return whether the package was stored into the db.
func (db *Database) HasPackage(name string) bool {
	var found bool = false
	db.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("package")).Cursor()
		for k, _ := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, _ = c.Next() {
			found = true
			return nil
		}
		return nil
	})
	return found
}

// Return a list of package names.
func (db *Database) GetPackageNames() []string {
	var list = []string{}
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
		bucket.ForEach(func(k, v []byte) error {
			list = append(list, string(k))
			return nil
		})
		return nil
	})
	return list
}

// Return a list of all packages.
func (db *Database) ListAllPackages() []*Package {
	var list []*Package
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
		bucket.ForEach(func(k, v []byte) error {
			pkg := &Package{}
			json.Unmarshal(v, &pkg)
			list = append(list, pkg)
			return nil
		})
		return nil
	})
	return list
}

// Add a package to the database.
func (db *Database) AddPackage(pkg *Package) error {
	encoded, err := json.Marshal(pkg)
	if err != nil {
		return nil
	}
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("package"))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(pkg.Name), encoded)
		if err != nil {
			return err
		}

		return nil
	})
}

// Remove a package from the database.
func (db *Database) RemovePackage(name string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(name))
	})
}
