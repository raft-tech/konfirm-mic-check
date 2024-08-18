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

package source

import (
	_ "embed"
	"io"
)

const (
	Kilobyte = 1024
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
)

var (
	//go:embed 4kb
	fourKiB []byte
)

type Source interface {
	io.Reader
}

func New(size int) Source {
	return &source{
		size: size,
	}
}

type source struct {
	pos  int
	rpos int
	size int
}

func (s *source) Read(p []byte) (n int, err error) {

	// Determine the read size
	n = s.size - s.pos
	if l := len(p); l < n {
		n = l
	}

	// Copy from fourKiB
	for i := 0; i < n; {

		// Determine the to write chunk
		m := len(fourKiB) - s.rpos
		if n2 := n - i; n2 < m {
			m = n2
		}
		chunk := fourKiB[s.rpos : s.rpos+m]

		// Copy to p and update positions
		m = copy(p[i:], chunk)
		s.pos += m
		i += m
		if s.rpos += m; s.rpos == len(fourKiB) {
			s.rpos = 0
		}
	}

	// Return EOF when size is met
	if s.pos == s.size {
		err = io.EOF
	}

	return
}
