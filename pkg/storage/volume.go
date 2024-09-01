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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
)

const Index = "index.json"

type Volume interface {
	NewInstance() (Instance, error)
	Instances() []Instance
	Trim() error
	Scrub() error
}

type VolumeOption interface {
	apply(vol *volume)
}

func New(basePath string, opt ...VolumeOption) (Volume, error) {

	// Validate basePath
	basePath = strings.TrimRight(basePath, "/\\")
	if finfo, err := os.Stat(basePath); errors.Is(err, fs.ErrNotExist) {
		if err = os.MkdirAll(basePath, 0755); err != nil {
			return nil, err
		}
	} else if !finfo.IsDir() {
		return nil, errors.New("basePath is not a directory")
	}

	vol := &volume{
		logger:       zap.NewNop(),
		basePath:     basePath,
		maxInstances: 5,
	}
	for _, o := range opt {
		o.apply(vol)
	}
	vol.logger.Info("volume instantiated", zap.String("basePath", vol.basePath))

	return vol, vol.loadIndex()
}

type volume struct {
	logger       *zap.Logger
	basePath     string
	maxInstances int
	index        map[string][]VolumeEntry
	instances    map[string]Instance
}

func (v *volume) loadIndex() error {

	// Load the index file if it exists
	logger := v.logger
	logger.Debug("loading index")
	v.index = make(map[string][]VolumeEntry)
	if buf, err := os.ReadFile(path.Join(v.basePath, Index)); err == nil {
		var ver version
		if err = json.NewDecoder(bytes.NewReader(buf)).Decode(&ver); err != nil {
			logger.Error("error reading index file", zap.Error(err))
			return err
		}
		if ver.Version == 1 {
			logger = logger.With(zap.Int("version", ver.Version))
			var idx v1
			dec := json.NewDecoder(bytes.NewReader(buf))
			dec.DisallowUnknownFields()
			if err = dec.Decode(&idx); err != nil {
				logger.Error("error decoding index file", zap.Error(err))
				return err
			}
			v.index = idx.Index
			logger.Info("index loaded")
		} else {
			logger.Error("unrecognized index version", zap.Int("version", ver.Version))
			return errors.New("unrecognized index version")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		logger.Error("error reading index", zap.Error(err))
		return err
	}

	// Create instances
	v.instances = make(map[string]Instance)
	for name, entries := range v.index {
		v.instances[name] = &instance{
			logger:   v.logger.With(zap.String("instance", name)),
			basePath: path.Join(v.basePath, name),
			entries:  entries,
			onAdd: func(entry VolumeEntry) {
				v.index[name] = append(v.index[name], entry)
				_ = v.writeIndex()
			},
		}
	}

	return v.Trim()
}

func (v *volume) writeIndex() error {
	v.logger.Debug("writing index file")
	var enc *json.Encoder
	if f, err := os.OpenFile(path.Join(v.basePath, Index), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err == nil {
		defer func() {
			if e := f.Sync(); e != nil {
				v.logger.Error("error syncing index file after write", zap.Error(err))
			}
			if e := f.Close(); e != nil {
				v.logger.Error("error closing index file after write", zap.Error(err))
			}
		}()
		enc = json.NewEncoder(f)
	} else {
		v.logger.Error("error writing index file", zap.Error(err))
		return err
	}
	if err := enc.Encode(index(v.index)); err != nil {
		v.logger.Error("error writing index", zap.Error(err))
		return err
	}
	v.logger.Info("successfully wrote index file")
	return nil
}

func (v *volume) NewInstance() (Instance, error) {

	logger := v.logger
	logger.Debug("creating new instance")

	// Get a unique instance name by sleeping until the next second on collisions
	now := time.Now().Unix()
	name := fmt.Sprintf("%d", now)
	if _, ok := v.index[name]; ok {
		logger.Debug("rate limiting instance creation to ensure unique instance names")
		time.Sleep(time.Until(time.Unix(now+1, 0)))
		name = fmt.Sprintf("%d", time.Now().Unix())
	}
	logger = logger.With(zap.String("instance", name))

	// Create the instance directory
	dirName := path.Join(v.basePath, name)
	logger.Debug("creating instance directory")
	if e := os.Mkdir(dirName, 0755); e != nil {
		logger.Error("error creating instance directory", zap.Error(e))
		return nil, e
	}
	logger.Info("created instance directory", zap.String("path", name))

	// Update the index
	v.index[name] = []VolumeEntry{}
	v.instances[name] = &instance{
		logger:   logger,
		basePath: dirName,
		entries:  nil,
		onAdd: func(entry VolumeEntry) {
			v.index[name] = append(v.index[name], entry)
			_ = v.writeIndex()
		},
	}
	logger.Info("created instance")

	_ = v.Trim()

	return v.instances[name], v.writeIndex()
}

func (v *volume) Instances() []Instance {
	instances := make([]Instance, 0, len(v.instances))
	for _, inst := range v.instances {
		instances = append(instances, inst)
	}
	return instances
}

func (v *volume) Trim() error {

	logger := v.logger
	if l := len(v.index); l <= v.maxInstances {
		logger.Info("skipping unnecessary trim operation", zap.Int("currentInstances", l), zap.Int("maxInstances", v.maxInstances))
		return nil
	} else {
		logger.Debug("starting trim operation", zap.Int("maxInstances", v.maxInstances))
	}

	var err []error
	var entries []string
	for dir := range v.index {
		entries = append(entries, dir)
	}
	slices.Sort(entries)                            // sort naturally
	entries = entries[:len(entries)-v.maxInstances] // ignore most recent
	for i := range entries {
		logger := logger.With(zap.String("instance", entries[i]))
		logger.Debug("trimming instance")
		if e := os.RemoveAll(path.Join(v.basePath, entries[i])); e == nil {
			logger.Info("trimmed instance")
			delete(v.index, entries[i])
			delete(v.instances, entries[i])
		} else {
			logger.Error("error trimming instance", zap.Error(e))
		}
	}

	return errors.Join(err...)
}

func (v *volume) Scrub() error {

	logger := v.logger

	// Perform a trim
	var err []error
	err = append(err, v.Trim())

	// Get all directory entries
	var dir []os.DirEntry
	if d, e := os.ReadDir(v.basePath); e == nil {
		dir = d
	} else {
		logger.Error("error reading basePath directory", zap.Error(e))
		err = append(err, e)
		return errors.Join(err...)
	}

	// Remove any unexpected files/dirs
	for i := range dir {
		entry := dir[i]
		if entry.IsDir() {
			if _, ok := v.index[entry.Name()]; !ok {
				logger := logger.With(zap.String("dir", entry.Name()))
				logger.Debug("scrubbing directory")
				if e := os.RemoveAll(path.Join(v.basePath, entry.Name())); e == nil {
					logger.Info("scrubbed directory")
				} else {
					logger.Error("error scrubbing directory", zap.Error(e))
					err = append(err, e)
				}
			}
		} else if n := entry.Name(); n != Index {
			logger := logger.With(zap.String("file", entry.Name()))
			logger.Debug("scrubbing file")
			if e := os.Remove(path.Join(v.basePath, n)); e == nil {
				logger.Info("scrubbed file")
			} else {
				logger.Error("error scrubbing file", zap.Error(e))
				err = append(err, e)
			}
		}
	}

	return errors.Join(err...)
}

func index(idx map[string][]VolumeEntry) v1 {
	return v1{
		version: version{
			Version: 1,
		},
		Index: idx,
	}
}

type version struct {
	Version int `json:"version"`
}

func v1Index(index map[string][]VolumeEntry) v1 {
	return v1{
		version: version{Version: 1},
		Index:   index,
	}
}

type v1 struct {
	version
	Index map[string][]VolumeEntry `json:"index"`
}

func WithMaxInstances(n int) VolumeOption {
	return maxInstancesOption{max: n}
}

type maxInstancesOption struct {
	max int
}

func (o maxInstancesOption) apply(vol *volume) {
	vol.maxInstances = o.max
}

func WithLogger(logger *zap.Logger) VolumeOption {
	return loggerOption{logger: logger}
}

type loggerOption struct {
	logger *zap.Logger
}

func (o loggerOption) apply(vol *volume) {
	vol.logger = o.logger
}
