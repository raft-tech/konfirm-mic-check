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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

func TestStorageCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage")
}

var _ = Describe("command", func() {

	var args []string

	Context("with report", func() {

		var report *types.Report

		It("runs", func() {
			Expect(report).NotTo(BeNil())
			Expect(report.SuiteSucceeded).To(BeTrue())
		})

		JustBeforeEach(func(ctx context.Context) {

			jsonFile := fmt.Sprintf("%s/%s", GinkgoT().TempDir(), "runs.json")

			cmd := New()
			cmd.SetArgs(append([]string{"--", "--ginkgo.json-report", jsonFile}, args...))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			Expect(cmd.ExecuteContext(ctx)).To(Succeed())

			var reports []types.Report
			if f, err := os.Open(jsonFile); err == nil {
				decoder := json.NewDecoder(f)
				Expect(decoder.Decode(&reports)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(reports).To(HaveLen(1))
			report = &reports[0]
		})
	})

	BeforeEach(func() {
		args = []string{
			"--konfirm.base-dir", GinkgoT().TempDir(),
			"tiny:1KiB",
		}
	})
})
