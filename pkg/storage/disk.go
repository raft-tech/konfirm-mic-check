//go:build !windows

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

import "golang.org/x/sys/unix"

func GetDisk(dir string) (Disk, error) {
	d := &disk{}
	err := unix.Statfs(dir, &d.Statfs_t)
	return d, err
}

type disk struct {
	unix.Statfs_t
}

func (d disk) TotalBytes() uint64 {
	return uint64(d.Bsize) * d.Blocks
}

func (d disk) AvailableBytes() uint64 {
	return uint64(d.Bsize) * d.Bfree
}
