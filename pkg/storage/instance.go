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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/pkg/storage/source"
)

var (
	UnexpectedSizeErr = errors.New("entry does not match expected size")
	MessageDigestErr  = errors.New("calculated message digest does not match expected digest")
)

type Instance interface {
	Name() string
	Add(name string, src source.Source) VolumeEntry
	Walk() ([]VolumeEntry, error)
}

type VolumeEntry struct {
	Path          string
	Digest        string
	ReadDuration  *time.Duration `json:"-"`
	WriteDuration *time.Duration
	Size          int64
	Error         error `json:"-"`
}

type instance struct {
	logger   *zap.Logger
	basePath string
	entries  []VolumeEntry
	onAdd    func(entry VolumeEntry)
}

func (inst *instance) Name() string {
	return path.Base(inst.basePath)
}

func (inst *instance) Add(name string, src source.Source) VolumeEntry {

	logger := inst.logger.With(zap.String("name", name))
	entry := VolumeEntry{Path: name}

	logger.Debug("opening new file")
	var file *os.File
	if f, e := os.OpenFile(path.Join(inst.basePath, entry.Path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644); e == nil {
		logger.Info("file opened")
		file = f
	} else {
		logger.Error("error opening file", zap.Error(e))
		entry.Error = e
		return entry
	}

	logger.Debug("writing to file")
	digest := sha256.New()
	start := time.Now()
	if n, e := file.ReadFrom(io.TeeReader(src, digest)); e == nil {
		t := time.Since(start)
		entry.WriteDuration = &t
		entry.Size = n
		entry.Digest = "sha256:" + hex.EncodeToString(digest.Sum(nil))
		logger.Info("successfully wrote to file", zap.Int64("bytes", n), zap.Durationp("duration", entry.WriteDuration))
	} else {
		logger.Error("error writing to file", zap.Error(e))
		entry.Error = e
		return entry
	}

	inst.entries = append(inst.entries, entry)
	inst.onAdd(entry)
	return entry
}

func (inst *instance) Walk() ([]VolumeEntry, error) {
	entries := make([]VolumeEntry, 0, len(inst.entries))
	digest := sha256.New()
	for i := range inst.entries {
		entry := VolumeEntry{
			Path:   inst.entries[i].Path,
			Digest: inst.entries[i].Digest,
			Size:   inst.entries[i].Size,
		}
		logger := inst.logger.With(zap.String("entry", entry.Path))

		logger.Debug("opening file")

		if f, err := os.OpenFile(path.Join(inst.basePath, entry.Path), os.O_RDONLY, 0); err == nil {
			logger.Info("starting file read")
			digest.Reset()
			start := time.Now()
			if n, e := f.WriteTo(digest); e == nil {
				t := time.Since(start)
				entry.ReadDuration = &t
				if n != entry.Size {
					entry.Error = UnexpectedSizeErr
					logger.Warn("volume entry size mismatch", zap.Int64("expectedSize", entry.Size), zap.Int64("actualSize", n))
				} else if digestStr := "sha256:" + hex.EncodeToString(digest.Sum(nil)); digestStr != entry.Digest {
					entry.Error = MessageDigestErr
					logger.Warn("volume entry bad message digest", zap.String("expectedDigest", entry.Digest), zap.String("actualDigest", digestStr))
				}
			} else {
				logger.Error("error reading file", zap.Error(e))
				entry.Error = e
			}
			if e := f.Close(); e != nil {
				logger.Error("error closing file", zap.Error(e))
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			logger.Warn("volume entry file not found")
			entry.Error = fs.ErrNotExist
		} else {
			logger.Error("error opening file", zap.Error(err))
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
