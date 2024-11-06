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
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/gstruct"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/raft-tech/konfirm-inspections/pkg/storage/source"
)

var logger *zap.Logger

func TestStorage(t *testing.T) {
	logger = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(GinkgoWriter),
		zapcore.LevelOf(zapcore.DebugLevel),
	))
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage")
}

var _ = Describe("Volumes", func() {

	Context("with a volume", func() {

		var vol Volume
		var basePath string

		It("creates new instances", func() {

			var inst Instance
			if i, err := vol.NewInstance(); err == nil {
				inst = i
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			if f, err := os.Stat(path.Join(basePath, inst.Name())); err == nil {
				Expect(f.IsDir()).To(BeTrue())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			if v, err := New(basePath, WithLogger(logger)); err == nil {
				Expect(v.Instances()).To(And(
					HaveLen(1),
					ContainElement(gcustom.MakeMatcher(func(i Instance) (bool, error) {
						return i != nil && i.Name() == inst.Name(), nil
					})),
				))
			} else {
				Expect(err).To(HaveOccurred())
			}
		})

		It("adds and validates entries", func() {

			var inst Instance
			if i, err := vol.NewInstance(); err == nil {
				inst = i
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			entry := inst.Add("test", source.New(1024))
			Expect(entry.Error).NotTo(HaveOccurred())
			Expect(inst.Walk()).To(And(
				HaveLen(1),
				ContainElement(gstruct.MatchAllFields(gstruct.Fields{
					"Path":          Equal(entry.Path),
					"Size":          Equal(entry.Size),
					"ReadDuration":  Not(BeNil()),
					"WriteDuration": BeNil(),
					"Digest":        Equal(entry.Digest),
					"Error":         BeNil(),
				}),
				)))
		})

		It("trims instances as they are created", func() {
			for i := 0; i < 4; i++ {
				_, err := vol.NewInstance()
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(vol.Instances()).To(HaveLen(3))
		})

		It("scrubs unexpected directories and files", func() {

			// Pollute the base path
			Expect(os.WriteFile(path.Join(basePath, "a-file"), []byte("This is an unexpected file"), 0644)).To(Succeed())
			Expect(os.MkdirAll(path.Join(basePath, "a-dir"), 0755)).To(Succeed())
			Expect(os.WriteFile(path.Join(basePath, "a-dir", "another-file"), []byte("A nested unexpected dir"), 0644)).To(Succeed())

			// Scrub
			Expect(vol.Scrub()).To(Succeed())

			// Validate the base path has been scrubbed
			var err error
			_, err = os.ReadFile(path.Join(basePath, "a-file"))
			Expect(err).To(MatchError(fs.ErrNotExist))
			_, err = os.ReadDir(path.Join(basePath, "a-dir"))
			Expect(err).To(MatchError(fs.ErrNotExist))
		})

		Context("with an instance", func() {

			var instanceName string
			var entry VolumeEntry

			It("Loads instances", func() {

				instances := vol.Instances()
				Expect(instances).To(HaveLen(1))
				Expect(instances[0].Name()).To(Equal(instanceName))

				entries, err := instances[0].Walk()
				Expect(err).NotTo(HaveOccurred())
				Expect(entries).To(HaveLen(1))
				Expect(entries[0]).To(gstruct.MatchAllFields(gstruct.Fields{
					"Path":          Equal(entry.Path),
					"Size":          Equal(entry.Size),
					"ReadDuration":  Not(BeNil()),
					"WriteDuration": BeNil(),
					"Digest":        Equal(entry.Digest),
					"Error":         BeNil(),
				}))
			})

			Context("and a bad size", func() {

				It("errors on the size", func() {
					entries, err := vol.Instances()[0].Walk()
					Expect(err).NotTo(HaveOccurred())
					Expect(entries).To(HaveLen(1))
					Expect(entries[0].Error).To(MatchError(UnexpectedSizeErr))
				})

				BeforeEach(func() {

					f, err := os.OpenFile(path.Join(basePath, Index), os.O_RDWR, 0)
					Expect(err).NotTo(HaveOccurred())

					var index v1
					Expect(json.NewDecoder(f).Decode(&index)).To(Succeed())
					index.Index[instanceName][0].Size += 2

					Expect(f.Truncate(0)).To(Succeed())
					Expect(f.Seek(0, 0)).To(Equal(int64(0)))
					Expect(json.NewEncoder(f).Encode(&index)).To(Succeed())
					Expect(f.Sync()).To(Succeed())
					Expect(f.Close()).To(Succeed())
				})
			})

			Context("and a bad digest", func() {

				It("errors on the size", func() {
					entries, err := vol.Instances()[0].Walk()
					Expect(err).NotTo(HaveOccurred())
					Expect(entries).To(HaveLen(1))
					Expect(entries[0].Error).To(MatchError(MessageDigestErr))
				})

				BeforeEach(func() {

					f, err := os.OpenFile(path.Join(basePath, Index), os.O_RDWR, 0)
					Expect(err).NotTo(HaveOccurred())

					var index v1
					Expect(json.NewDecoder(f).Decode(&index)).To(Succeed())
					index.Index[instanceName][0].Digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

					Expect(f.Truncate(0)).To(Succeed())
					Expect(f.Seek(0, 0)).To(Equal(int64(0)))
					Expect(json.NewEncoder(f).Encode(&index)).To(Succeed())
					Expect(f.Sync()).To(Succeed())
					Expect(f.Close()).To(Succeed())
				})
			})

			BeforeEach(func() {

				instanceName = fmt.Sprintf("%d", time.Now().Unix())
				Expect(os.Mkdir(path.Join(basePath, instanceName), 0755)).To(Succeed())
				entry.Path = "test"
				entry.Size = 512

				f, err := os.OpenFile(path.Join(basePath, instanceName, entry.Path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())
				digest := sha256.New()
				reader := io.TeeReader(source.New(entry.Size), digest)
				start := time.Now()
				size, err := f.ReadFrom(reader)
				t := time.Since(start)
				entry.WriteDuration = &t
				entry.Digest = "sha256:" + hex.EncodeToString(digest.Sum(nil))
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(entry.Size))

				f, err = os.OpenFile(path.Join(basePath, Index), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.NewEncoder(f).Encode(v1Index(map[string][]VolumeEntry{
					instanceName: {entry},
				}))).To(Succeed())
			})
		})

		BeforeEach(func() {
			basePath = GinkgoT().TempDir()
		})

		JustBeforeEach(func() {
			var err error
			vol, err = New(basePath, WithLogger(logger), WithMaxInstances(3))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	AfterEach(func() {
		Expect(logger.Sync()).To(Succeed())
	})
})
