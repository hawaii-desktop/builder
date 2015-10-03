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

package main

// Variant properties map.
type VariantMap map[string]interface{}

// Returns whether the map contains the item.
func (v VariantMap) Contains(k string) bool {
	if _, ok := v[k]; ok {
		return true
	}
	return false
}

// Returns a string property.
func (v VariantMap) GetString(k string, d string) string {
	if v.Contains(k) {
		return v[k].(string)
	}
	return d
}

// Returns an int property.
func (v VariantMap) GetInt(k string, d int) int {
	if v.Contains(k) {
		return v[k].(int)
	}
	return d
}

// Returns an uint property.
func (v VariantMap) GetUint(k string, d uint) uint {
	if v.Contains(k) {
		return v[k].(uint)
	}
	return d
}

// Returns an int32 property.
func (v VariantMap) GetInt32(k string, d int32) int32 {
	if v.Contains(k) {
		return v[k].(int32)
	}
	return d
}

// Returns an uint32 property.
func (v VariantMap) GetUint32(k string, d uint32) uint32 {
	if v.Contains(k) {
		return v[k].(uint32)
	}
	return d
}

// Returns an int32 property.
func (v VariantMap) GetInt64(k string, d int64) int64 {
	if v.Contains(k) {
		return v[k].(int64)
	}
	return d
}

// Returns an uint64 property.
func (v VariantMap) GetUint64(k string, d uint64) uint64 {
	if v.Contains(k) {
		return v[k].(uint64)
	}
	return d
}

// Returns a bool property.
func (v VariantMap) GetBool(k string, d bool) bool {
	if v.Contains(k) {
		return v[k].(bool)
	}
	return d
}
