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
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
)

type Chroot struct {
	// Release (fedora, epel, ...)
	OsRelease string `json:"release"`
	// Version (22, 23, rawhide, ...)
	OsVersion string `json:"version"`
	// Architecture (x86_64, i386, armhfp, ...)
	Architecture string `json:"arch"`
	// Is it active?
	Active bool `json:"active"`
}

// Return a list of all chroots.
func (db *Database) ListAllChroots() []*Chroot {
	var list []*Chroot
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("chroot"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			chroot := &Chroot{}
			json.Unmarshal(v, &chroot)
			list = append(list, chroot)
			return nil
		})
		return nil
	})
	return list
}

// Return a list of all active chroots.
func (db *Database) ListActiveChroots() []*Chroot {
	var list []*Chroot
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("chroot"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			chroot := &Chroot{}
			json.Unmarshal(v, &chroot)
			if chroot.Active {
				list = append(list, chroot)
			}
			return nil
		})
		return nil
	})
	return list
}

// Return an instance of the chroot with release, version and
// architecture from the database.
func (db *Database) GetChroot(release, version, arch string) *Chroot {
	name := fmt.Sprintf("%s-%s-%s", release, version, arch)

	var chroot *Chroot = nil
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("chroot"))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			json.Unmarshal(v, &chroot)
			return nil
		}
		return nil
	})
	return chroot
}

// Add a chroot to the database.
func (db *Database) AddChroot(chroot *Chroot) error {
	name := fmt.Sprintf("%s-%s-%s", chroot.OsRelease, chroot.OsVersion, chroot.Architecture)

	encoded, err := json.Marshal(chroot)
	if err != nil {
		return err
	}

	err = db.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("chroot"))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(name), encoded)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

// Remove a chroot from the database.
func (db *Database) RemoveChroot(release, version, arch string) error {
	name := fmt.Sprintf("%s-%s-%s", release, version, arch)

	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("chroot"))
		if bucket == nil {
			return nil
		}

		// Unmarshal the package
		chroot := &Chroot{}
		c := bucket.Cursor()
		for k, v := c.Seek([]byte(name)); bytes.Equal(k, []byte(name)); k, v = c.Next() {
			if err := json.Unmarshal(v, &chroot); err != nil {
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

// Iterate the chroot list.
func (db *Database) ForEachChroot(f func(chroot *Chroot)) {
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("chroot"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(k, v []byte) error {
			var chroot *Chroot
			err := json.Unmarshal(v, &chroot)
			if err != nil {
				return err
			}

			f(chroot)

			return nil
		})

		return nil
	})
}

// Textual representation of name of this chroot.
func (c *Chroot) Name() string {
	return fmt.Sprintf("%s-%s-%s", c.OsRelease, c.OsVersion, c.Architecture)
}

// Textual representation of name and release.
func (c *Chroot) NameRelease() string {
	return fmt.Sprintf("%s-%s", c.OsRelease, c.OsVersion)
}

// Human representation of name and release.
func (c *Chroot) NameReleaseHuman() string {
	return fmt.Sprintf("%s %s", c.OsRelease, c.OsVersion)
}
