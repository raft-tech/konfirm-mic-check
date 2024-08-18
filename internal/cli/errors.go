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

package cli

import (
	"errors"
	"fmt"
)

type ExitError interface {
	error
	ExitCode() int
}

func ErrorF(code int, msg string, args ...any) ExitError {
	return xerror{
		error: errors.New(fmt.Sprintf(msg, args...)),
		code:  code,
	}
}

func Wrap(code int, err error) ExitError {
	return xerror{
		error: err,
		code:  code,
	}
}

type xerror struct {
	error
	code int
}

func (e xerror) ExitCode() int {
	return e.code
}
