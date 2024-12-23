//go:build inspection

/*
 * Copyright (c) 2024 Raft, LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package storage

import (
	"time"
)

type observer struct {
	count  int
	sum    time.Duration
	errors int
}

func (o *observer) Observe(value time.Duration) {
	o.count++
	o.sum += value
}

func (o *observer) ObserveError(err error) {
	if err != nil {
		o.errors++
	}
}

func (o *observer) Average() time.Duration {
	return o.sum / time.Duration(o.count)
}
