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
	"errors"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	//go:embed 4kb
	fourKiB []byte
)

type Source interface {
	io.Reader
}

func New(size int64) Source {
	return &source{
		size: size,
	}
}

type source struct {
	pos  int64
	rpos int
	size int64
}

func (s *source) Read(p []byte) (n int, err error) {

	// Determine the read size
	n = len(p)
	if remaining := s.size - s.pos; remaining < int64(n) {
		n = int(remaining)
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
		s.pos += int64(m)
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

type Spec interface {
	Name() string
	Describe() string
	Size() int64
	Generate() Source
}

var InvalidSizeFormatErr = errors.New("sizes must be formated as an integer followed by an optional unit (e.g., 4KiB)")
var ExceedsMaxSizeErr = errors.New("size exceeds maximum size")

// NewSpec creates a Spec based on the provided description/size. If desc and size are both
// defined, desc is used as the Spec name and size is parsed to determine the Int and Int64 values.
// Optionally, the size may be embedded in a colon-delineated description (e.g., medium:256Ki)
// and size left an empty string. This option is useful when parsing command-line arguments.
//
// Supported size units are the same as resource.Quantity. If no unit is specified, the unit is
// assumed to be Bytes. InvalidSizeFormatErr is returned if the specified size is not in a
// recognized format. ExceedsMaxSizeErr is returned if the specified size is greater than the max
// int64 value when converted to bytes.
//
// If no error is return, the returned Spec will be valid.
//
// See Source.
func NewSpec(desc, size string) (Spec, error) {

	// Optionally split desc at ':' to set size
	name := desc
	if size == "" {
		if d := strings.SplitN(desc, ":", 2); len(d) == 2 {
			name = d[0]
			size = d[1]
		}
	}

	var qty resource.Quantity
	if q, err := resource.ParseQuantity(size); err == nil {
		qty = q
	} else {
		return nil, errors.Join(InvalidSizeFormatErr, err)
	}

	if i, ok := qty.AsInt64(); ok {
		return spec{
			name: name,
			desc: desc,
			size: i,
		}, nil
	} else {
		return nil, ExceedsMaxSizeErr
	}
}

type spec struct {
	name string
	desc string
	size int64
}

func (s spec) Name() string {
	return s.name
}

func (s spec) Describe() string {
	return s.desc
}

func (s spec) Size() int64 {
	return s.size
}

func (s spec) Generate() Source {
	return New(s.size)
}
