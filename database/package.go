/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
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
		bucket := tx.Bucket([]byte("package"))
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

// Return a list of package names.
func (db *Database) GetPackageNames() []string {
	var list = []string{}
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
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

// Return a list of all packages.
func (db *Database) ListAllPackages() []*Package {
	var list []*Package
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
		if bucket == nil {
			return nil
		}

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

// Return a package from the database.
func (db *Database) GetPackage(name string) *Package {
	var pkg *Package = nil
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			json.Unmarshal(v, &pkg)
			return nil
		}
		return nil
	})
	return pkg
}

// Add a package to the database.
func (db *Database) AddPackage(pkg *Package) error {
	encoded, err := json.Marshal(pkg)
	if err != nil {
		return err
	}

	err = db.db.Update(func(tx *bolt.Tx) error {
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

	if err == nil {
		// Add the architectures as supported
		err = db.SaveArchitectures(pkg.Architectures...)
		if err != nil {
			return err
		}
	}

	return err
}

// Remove a package from the database.
func (db *Database) RemovePackage(name string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
		if bucket == nil {
			return nil
		}

		// Unmarshal the package
		pkg := &Package{}
		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			if err := json.Unmarshal(v, &pkg); err != nil {
				return err
			}
		}

		// Delete the bucket
		if err := bucket.Delete([]byte(name)); err != nil {
			return err
		}

		// Remove these architectures if they are not referenced
		// by any other package or image
		if err := db.RemoveArchitectures(pkg.Architectures...); err != nil {
			return err
		}

		return nil
	})
}

// Iterate the packages list.
func (db *Database) ForEachPackage(f func(pkg *Package)) {
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("package"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			var pkg *Package
			err := json.Unmarshal(v, &pkg)
			if err != nil {
				return err
			}

			f(pkg)

			return nil
		})

		return nil
	})
}
