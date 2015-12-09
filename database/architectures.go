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
	"encoding/json"
	"github.com/boltdb/bolt"
	"github.com/hawaii-desktop/builder/utils"
)

// Return a list of all the architectures.
func (db *Database) ListArchitectures() []string {
	var archs []string
	db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket == nil {
			return nil
		}

		v := bucket.Get([]byte("archs"))
		err := json.Unmarshal(v, &archs)
		if err != nil {
			return err
		}

		return nil
	})
	return archs
}

// Save the list of supported architectures.
func (db *Database) SaveArchitectures(archsList ...string) error {
	// Make sure the list has only unique architectures
	archs := utils.DeduplicateStringSlice(archsList)

	return db.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("data"))
		if err != nil {
			return err
		}

		// Retrieve the architectures that we currently have
		var savedArchs []string
		v := bucket.Get([]byte("archs"))
		if v != nil {
			err = json.Unmarshal(v, &savedArchs)
			if err != nil {
				return err
			}
		}

		// Add the architectures only if needed
		changed := false
		for _, a := range archs {
			if !utils.StringSliceContains(savedArchs, a) {
				savedArchs = append(savedArchs, a)
				changed = true
			}
		}

		// Store the list if it has changed
		if changed {
			encoded, err := json.Marshal(savedArchs)
			if err != nil {
				return err
			}
			err = bucket.Put([]byte("archs"), encoded)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Remove an architecture if it's not referenced anymore
// by any package or image.
func (db *Database) RemoveArchitectures(archsList ...string) error {
	// Make sure the list has only unique architectures
	archs := utils.DeduplicateStringSlice(archsList)

	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("data"))
		if bucket == nil {
			return nil
		}

		// Retrieve the architectures that we currently have
		var savedArchs []string
		v := bucket.Get([]byte("archs"))
		err := json.Unmarshal(v, &savedArchs)
		if err != nil {
			return err
		}

		// Determine what architectures are unreferenced
		var unreferenced []string
		for _, arch := range archs {
			referenced := false

			db.ForEachPackage(func(pkg *Package) {
				if utils.StringSliceContains(pkg.Architectures, arch) {
					referenced = true
				}
			})

			db.ForEachImage(func(img *Image) {
				if utils.StringSliceContains(img.Architectures, arch) {
					referenced = true
				}
			})

			if !referenced {
				unreferenced = append(unreferenced, arch)
			}
		}

		// Remove unreferenced architectures
		changed := false
		for i, a := range savedArchs {
			for _, arch := range unreferenced {
				if a == arch {
					savedArchs = append(savedArchs[:i], savedArchs[i+1:]...)
					changed = true
				}
			}
		}

		// Save the list of architectures if changed
		if changed {
			encoded, err := json.Marshal(savedArchs)
			if err != nil {
				return err
			}
			err = bucket.Put([]byte("archs"), encoded)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
