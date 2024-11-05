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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage")
}

var _ = Describe("Source", func() {

	It("1.5 KiB", func() {

		// Given
		fixture := New(1536)

		expected := make([]byte, 1536)
		Expect(copy(expected, fourKiB)).To(Equal(1536))

		expectedSHA256 := sha256.New()
		Expect(expectedSHA256.Write(expected)).To(Equal(1536))
		expectedString := hex.EncodeToString(expectedSHA256.Sum(nil))

		// When
		digest := sha256.New()
		reader := io.TeeReader(fixture, digest)
		actual := bytes.NewBuffer(make([]byte, 0, 1536))
		var err error
		for buf := make([]byte, 2048); err == nil; {
			var n int
			if n, err = reader.Read(buf); n != 2048 {
				Expect(err).To(MatchError(io.EOF))
			}
			Expect(actual.Write(buf[:n])).To(Equal(n))
		}

		// Then
		Expect(actual.Bytes()).To(HaveLen(1536))
		Expect(actual.Bytes()).To(Equal(expected))
		Expect(hex.EncodeToString(digest.Sum(nil))).To(Equal(expectedString))
	})

	It("4.5 KiB", func() {

		// Given
		fixture := New(6144)

		expected := make([]byte, 6144)
		Expect(copy(expected, fourKiB)).To(Equal(4096))
		Expect(copy(expected[4096:], fourKiB)).To(Equal(2048))

		expectedSHA256 := sha256.New()
		Expect(expectedSHA256.Write(expected)).To(Equal(6144))
		expectedString := hex.EncodeToString(expectedSHA256.Sum(nil))

		// When
		digest := sha256.New()
		reader := io.TeeReader(fixture, digest)
		actual := bytes.NewBuffer(make([]byte, 0, 6144))
		var err error
		for buf := make([]byte, 513); err == nil; {
			var n int
			if n, err = reader.Read(buf); n != 513 {
				Expect(err).To(MatchError(io.EOF))
			}
			Expect(actual.Write(buf[:n])).To(Equal(n))
		}

		// Then
		Expect(actual.Bytes()).To(HaveLen(6144))
		Expect(actual.Bytes()).To(Equal(expected))
		Expect(hex.EncodeToString(digest.Sum(nil))).To(Equal(expectedString))
	})

	It("10 KiB", func() {

		// Given
		fixture := New(10240)

		expected := make([]byte, 10240)
		Expect(copy(expected, fourKiB)).To(Equal(4096))
		Expect(copy(expected[4096:], fourKiB)).To(Equal(4096))
		Expect(copy(expected[8192:], fourKiB)).To(Equal(2048))

		expectedSHA256 := sha256.New()
		Expect(expectedSHA256.Write(expected)).To(Equal(10240))
		expectedString := hex.EncodeToString(expectedSHA256.Sum(nil))

		// When
		digest := sha256.New()
		reader := io.TeeReader(fixture, digest)
		actual := bytes.NewBuffer(make([]byte, 0, 492))
		var err error
		for buf := make([]byte, 492); err == nil; {
			var n int
			if n, err = reader.Read(buf); n != 492 {
				Expect(err).To(MatchError(io.EOF))
			}
			Expect(actual.Write(buf[:n])).To(Equal(n))
		}

		// Then
		Expect(actual.Bytes()).To(HaveLen(10240))
		Expect(actual.Bytes()).To(Equal(expected))
		Expect(hex.EncodeToString(digest.Sum(nil))).To(Equal(expectedString))
	})
})

var _ = FDescribe("SourceSpec", func() {

	DescribeTable("Parses specs as expected", func(desc string, size string, expectedName string, expectedSize int64) {
		spec, err := NewSpec(desc, size)
		Expect(err).NotTo(HaveOccurred())
		Expect(spec.Describe()).To(Equal(desc))
		Expect(spec.Name()).To(Equal(expectedName))
		Expect(spec.Size()).To(Equal(expectedSize))
	},
		Entry("Whole Ki", "medium:4Ki", "", "medium", int64(4096)),
		Entry("Fractional Ki", "small:0.5Ki", "", "small", int64(512)),
		Entry("Bytes", "tiny:128", "", "tiny", int64(128)),
		Entry("Split", "large", "2.5G", "large", int64(2_500_000_000)),
	)
})
